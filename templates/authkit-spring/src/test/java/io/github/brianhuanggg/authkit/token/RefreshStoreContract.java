package io.github.brianhuanggg.authkit.token;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.security.SecureRandom;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.ArrayList;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.Callable;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import org.junit.jupiter.api.Test;

/**
 * Conformance suite for every {@link RefreshStore}, like Go's {@code storetest}.
 * Implementations supply a store and a user valid for it (the Postgres table
 * has a foreign key to {@code users}).
 */
interface RefreshStoreContract {

    RefreshStore store();

    UUID newUser();

    private static byte[] hash() {
        byte[] b = new byte[32];
        new SecureRandom().nextBytes(b);
        return b;
    }

    private static Instant in(long hours) {
        return Instant.now().plus(hours, ChronoUnit.HOURS).truncatedTo(ChronoUnit.MICROS);
    }

    private static Refresh fresh(UUID user, UUID family) {
        return new Refresh(hash(), user, family, null, in(1), in(24));
    }

    @Test
    default void createFind() {
        Refresh want = fresh(newUser(), UUID.randomUUID());
        store().create(want);
        Refresh got = store().find(want.idHash()).orElseThrow();
        assertThat(got.idHash()).isEqualTo(want.idHash());
        assertThat(got.userId()).isEqualTo(want.userId());
        assertThat(got.familyId()).isEqualTo(want.familyId());
        assertThat(got.usedAt()).isNull();
        assertThat(got.expiresAt()).isEqualTo(want.expiresAt());
        assertThat(got.familyExpiresAt()).isEqualTo(want.familyExpiresAt());
    }

    @Test
    default void findMissing() {
        assertThat(store().find(hash())).isEmpty();
    }

    @Test
    default void createDuplicate() {
        Refresh r = fresh(newUser(), UUID.randomUUID());
        store().create(r);
        assertThatThrownBy(() -> store().create(r)).isInstanceOf(DuplicateRefreshException.class);
    }

    @Test
    default void rotate() {
        UUID user = newUser();
        UUID family = UUID.randomUUID();
        Refresh old = fresh(user, family);
        Refresh next = fresh(user, family);
        store().create(old);
        Instant at = Instant.now().truncatedTo(ChronoUnit.MICROS);

        assertThat(store().rotate(old.idHash(), at, next)).isTrue();
        assertThat(store().find(old.idHash()).orElseThrow().usedAt()).isEqualTo(at);
        Refresh stored = store().find(next.idHash()).orElseThrow();
        assertThat(stored.usedAt()).isNull();
        assertThat(stored.familyId()).isEqualTo(family);

        Refresh again = fresh(user, family);
        assertThat(store().rotate(old.idHash(), at, again)).as("second rotate").isFalse();
        assertThat(store().find(again.idHash())).as("losing rotate stores nothing").isEmpty();
        assertThat(store().rotate(hash(), at, fresh(user, family))).as("missing token").isFalse();
    }

    /** A failed insert must not consume the old token, so the client can retry. */
    @Test
    default void rotateIsAllOrNothing() {
        UUID user = newUser();
        UUID family = UUID.randomUUID();
        Refresh old = fresh(user, family);
        Refresh taken = fresh(user, family);
        store().create(old);
        store().create(taken);

        Refresh collides = new Refresh(taken.idHash().clone(), user, family, null, in(1), in(24));
        assertThatThrownBy(() -> store().rotate(old.idHash(), Instant.now(), collides))
                .isInstanceOf(DuplicateRefreshException.class);
        assertThat(store().find(old.idHash()).orElseThrow().usedAt())
                .as("old token consumed although its replacement was not stored").isNull();
    }

    /** Exactly one concurrent caller may win the rotation. */
    @Test
    default void rotateIsAtomic() throws Exception {
        UUID user = newUser();
        UUID family = UUID.randomUUID();
        Refresh old = fresh(user, family);
        store().create(old);

        int callers = 20;
        CountDownLatch go = new CountDownLatch(1);
        ExecutorService pool = Executors.newFixedThreadPool(callers);
        try {
            List<Future<Boolean>> results = new ArrayList<>();
            for (int i = 0; i < callers; i++) {
                Callable<Boolean> call = () -> {
                    go.await();
                    return store().rotate(old.idHash(), Instant.now(), fresh(user, family));
                };
                results.add(pool.submit(call));
            }
            go.countDown();
            int wins = 0;
            for (Future<Boolean> f : results) {
                if (f.get()) {
                    wins++;
                }
            }
            assertThat(wins).as("winners").isEqualTo(1);
        } finally {
            pool.shutdownNow();
        }
    }

    @Test
    default void deleteExpired() {
        UUID user = newUser();
        Refresh live = fresh(user, UUID.randomUUID());
        Refresh dead = new Refresh(hash(), user, UUID.randomUUID(), null, in(-1), in(24));
        store().create(live);
        store().create(dead);
        assertThat(store().deleteExpired(Instant.now())).isGreaterThanOrEqualTo(1);
        assertThat(store().find(dead.idHash())).isEmpty();
        assertThat(store().find(live.idHash())).isPresent();
    }

    @Test
    default void deleteFamily() {
        UUID user = newUser();
        UUID family = UUID.randomUUID();
        Refresh a = fresh(user, family);
        Refresh b = fresh(user, family);
        Refresh other = fresh(user, UUID.randomUUID());
        store().create(a);
        store().create(b);
        store().create(other);

        store().deleteFamily(family);
        assertThat(store().find(a.idHash())).isEmpty();
        assertThat(store().find(b.idHash())).isEmpty();
        assertThat(store().find(other.idHash())).as("other family").isPresent();
        store().deleteFamily(UUID.randomUUID()); // unknown family is a no-op
    }

    @Test
    default void doesNotAliasInput() {
        Refresh r = fresh(newUser(), UUID.randomUUID());
        byte[] orig = r.idHash().clone();
        store().create(r);
        r.idHash()[0] ^= (byte) 0xff;
        assertThat(store().find(orig)).isPresent();
    }
}
