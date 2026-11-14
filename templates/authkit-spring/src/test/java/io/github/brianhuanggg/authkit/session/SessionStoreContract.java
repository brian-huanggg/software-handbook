package io.github.brianhuanggg.authkit.session;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.security.SecureRandom;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.UUID;
import org.junit.jupiter.api.Test;

/**
 * Conformance suite for every {@link SessionStore}, like Go's {@code storetest}.
 * Implementations supply a store and a user ID valid for it (the Postgres
 * table has a foreign key to {@code users}).
 */
interface SessionStoreContract {

    SessionStore store();

    UUID newUser();

    private static byte[] hash() {
        byte[] b = new byte[32];
        new SecureRandom().nextBytes(b);
        return b;
    }

    private static Instant in(long hours) {
        return Instant.now().plus(hours, ChronoUnit.HOURS).truncatedTo(ChronoUnit.MICROS);
    }

    @Test
    default void createGet() {
        Session want = new Session(hash(), newUser(), in(1));
        store().create(want);
        Session got = store().find(want.idHash()).orElseThrow();
        assertThat(got.idHash()).isEqualTo(want.idHash());
        assertThat(got.userId()).isEqualTo(want.userId());
        assertThat(got.expiresAt()).isEqualTo(want.expiresAt());
    }

    @Test
    default void getMissing() {
        assertThat(store().find(hash())).isEmpty();
    }

    @Test
    default void createDuplicate() {
        Session s = new Session(hash(), newUser(), in(1));
        store().create(s);
        assertThatThrownBy(() -> store().create(s)).isInstanceOf(DuplicateSessionException.class);
    }

    @Test
    default void delete() {
        Session s = new Session(hash(), newUser(), in(1));
        store().create(s);
        store().delete(s.idHash());
        assertThat(store().find(s.idHash())).isEmpty();
        store().delete(s.idHash()); // second delete is a no-op
    }

    @Test
    default void returnsExpiredRows() {
        Session s = new Session(hash(), newUser(), in(-1));
        store().create(s);
        assertThat(store().find(s.idHash())).as("store leaves expiry to SessionService").isPresent();
    }

    @Test
    default void deleteExpired() {
        UUID user = newUser();
        Session live = new Session(hash(), user, in(1));
        Session dead = new Session(hash(), user, in(-1));
        store().create(live);
        store().create(dead);
        assertThat(store().deleteExpired(Instant.now())).isGreaterThanOrEqualTo(1);
        assertThat(store().find(dead.idHash())).isEmpty();
        assertThat(store().find(live.idHash())).isPresent();
    }

    @Test
    default void doesNotAliasInput() {
        Session s = new Session(hash(), newUser(), in(1));
        byte[] orig = s.idHash().clone();
        store().create(s);
        s.idHash()[0] ^= (byte) 0xff; // caller mutates its array after create
        Session got = store().find(orig).orElseThrow();
        assertThat(got.idHash()).isEqualTo(orig);
    }
}
