package io.github.brianhuanggg.authkit.ratelimit;

import io.github.bucket4j.Bandwidth;
import io.github.bucket4j.Bucket;
import io.github.bucket4j.ConsumptionProbe;
import io.github.bucket4j.TimeMeter;
import java.time.Duration;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.LongSupplier;

/**
 * An in-memory token bucket per key: each key may spend {@code burst}
 * requests at once and regains one every {@code every}. It is per process, so
 * with several API replicas the effective limit is multiplied; move it to a
 * shared store (for example Redis) when you scale out.
 */
public final class Limiter {

    private static final long SWEEP_EVERY_NANOS = Duration.ofMinutes(1).toNanos();

    /** @param retryAfter how long until the next token; zero when allowed */
    public record Decision(boolean allowed, Duration retryAfter) {
        static final Decision OK = new Decision(true, Duration.ZERO);
    }

    private final Duration every;
    private final int burst;
    private final LongSupplier nanos;
    private final TimeMeter meter;
    private final ConcurrentHashMap<String, Bucket> buckets = new ConcurrentHashMap<>();
    private final AtomicLong nextSweep;

    public Limiter(Duration every, int burst) {
        this(every, burst, System::nanoTime);
    }

    /** Test hook: a time source other than the system clock. */
    public Limiter(Duration every, int burst, LongSupplier nanos) {
        if (every.isNegative() || every.isZero() || burst <= 0) {
            throw new IllegalArgumentException("Limiter needs every > 0 and burst > 0");
        }
        this.every = every;
        this.burst = burst;
        this.nanos = nanos;
        this.meter = new TimeMeter() {
            @Override
            public long currentTimeNanos() {
                return nanos.getAsLong();
            }

            @Override
            public boolean isWallClockBased() {
                return false;
            }
        };
        this.nextSweep = new AtomicLong(nanos.getAsLong() + SWEEP_EVERY_NANOS);
    }

    /** Spends one token for {@code key}. When none is left, says how long until the next one. */
    public Decision allow(String key) {
        sweep();
        ConsumptionProbe[] result = new ConsumptionProbe[1];
        // compute() holds the key's lock, so a sweep cannot drop a bucket mid-spend.
        buckets.compute(key, (k, existing) -> {
            Bucket bucket = existing != null ? existing : newBucket();
            result[0] = bucket.tryConsumeAndReturnRemaining(1);
            return bucket;
        });
        ConsumptionProbe probe = result[0];
        return probe.isConsumed() ? Decision.OK : new Decision(false, Duration.ofNanos(probe.getNanosToWaitForRefill()));
    }

    private Bucket newBucket() {
        return Bucket.builder()
                .addLimit(Bandwidth.builder().capacity(burst).refillGreedy(1, every).build())
                .withCustomTimePrecision(meter)
                .build();
    }

    /**
     * Drops buckets that have refilled completely: they behave exactly like a
     * missing bucket, so memory stays proportional to recently active keys.
     */
    private void sweep() {
        long now = nanos.getAsLong();
        long due = nextSweep.get();
        if (now - due < 0 || !nextSweep.compareAndSet(due, now + SWEEP_EVERY_NANOS)) {
            return;
        }
        for (String key : buckets.keySet()) {
            buckets.computeIfPresent(key, (k, b) -> b.getAvailableTokens() >= burst ? null : b);
        }
    }

    int size() {
        return buckets.size();
    }
}
