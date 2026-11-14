package io.github.brianhuanggg.authkit.ratelimit;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.time.Duration;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.atomic.AtomicLong;
import java.util.ArrayList;
import java.util.List;
import org.junit.jupiter.api.Test;

class LimiterTest {

    final AtomicLong now = new AtomicLong(1_000_000_000L);

    private void advance(Duration d) {
        now.addAndGet(d.toNanos());
    }

    @Test
    void allowsBurstThenRefusesWithRetryAfter() {
        Limiter l = new Limiter(Duration.ofSeconds(3), 3, now::get);
        for (int i = 0; i < 3; i++) {
            assertThat(l.allow("k").allowed()).as("request %d", i).isTrue();
        }
        Limiter.Decision d = l.allow("k");
        assertThat(d.allowed()).isFalse();
        assertThat(d.retryAfter()).isEqualTo(Duration.ofSeconds(3));
    }

    @Test
    void refillsOneTokenPerInterval() {
        Limiter l = new Limiter(Duration.ofSeconds(3), 2, now::get);
        l.allow("k");
        l.allow("k");
        assertThat(l.allow("k").allowed()).isFalse();

        advance(Duration.ofSeconds(1));
        Limiter.Decision d = l.allow("k");
        assertThat(d.allowed()).isFalse();
        assertThat(d.retryAfter()).as("2s of the 3s are left").isEqualTo(Duration.ofSeconds(2));

        advance(Duration.ofSeconds(2));
        assertThat(l.allow("k").allowed()).isTrue();
        assertThat(l.allow("k").allowed()).as("only one token came back").isFalse();
    }

    @Test
    void neverRefillsPastTheBurst() {
        Limiter l = new Limiter(Duration.ofSeconds(1), 2, now::get);
        advance(Duration.ofHours(1));
        assertThat(l.allow("k").allowed()).isTrue();
        assertThat(l.allow("k").allowed()).isTrue();
        assertThat(l.allow("k").allowed()).isFalse();
    }

    @Test
    void keysAreIndependent() {
        Limiter l = new Limiter(Duration.ofSeconds(30), 1, now::get);
        assertThat(l.allow("a").allowed()).isTrue();
        assertThat(l.allow("a").allowed()).isFalse();
        assertThat(l.allow("b").allowed()).isTrue();
    }

    @Test
    void sweepDropsFullyRefilledBucketsOnly() {
        Limiter l = new Limiter(Duration.ofSeconds(30), 3, now::get);
        l.allow("idle"); // 2 of 3 left: full again after 30s
        l.allow("busy");
        l.allow("busy");
        l.allow("busy"); // 0 of 3 left: full again after 90s
        assertThat(l.size()).isEqualTo(2);

        advance(Duration.ofSeconds(65)); // past the one-minute sweep interval
        l.allow("other"); // triggers the sweep
        assertThat(l.size()).as("idle dropped; busy and other kept").isEqualTo(2);
        assertThat(l.allow("idle").allowed()).as("a dropped bucket starts full").isTrue();
    }

    @Test
    void concurrentCallersNeverExceedTheBurst() throws Exception {
        Limiter l = new Limiter(Duration.ofHours(1), 10, now::get);
        int callers = 50;
        CountDownLatch go = new CountDownLatch(1);
        ExecutorService pool = Executors.newFixedThreadPool(callers);
        try {
            List<Future<Boolean>> results = new ArrayList<>();
            for (int i = 0; i < callers; i++) {
                results.add(pool.submit(() -> {
                    go.await();
                    return l.allow("k").allowed();
                }));
            }
            go.countDown();
            int allowed = 0;
            for (Future<Boolean> f : results) {
                if (f.get()) {
                    allowed++;
                }
            }
            assertThat(allowed).isEqualTo(10);
        } finally {
            pool.shutdownNow();
        }
    }

    @Test
    void rejectsBadConfig() {
        assertThatThrownBy(() -> new Limiter(Duration.ZERO, 1)).isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> new Limiter(Duration.ofSeconds(1), 0)).isInstanceOf(IllegalArgumentException.class);
    }
}
