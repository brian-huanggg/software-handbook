package io.github.brianhuanggg.authkit.token;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import io.github.brianhuanggg.authkit.session.SessionService;
import java.time.Duration;
import java.time.Instant;
import java.util.UUID;
import org.junit.jupiter.api.Test;

class RefreshTokenServiceTest {

    static final UUID USER = UUID.fromString("22222222-2222-2222-2222-222222222222");

    final JwtServiceTest.MutableClock clock = new JwtServiceTest.MutableClock();
    final InMemoryRefreshStore store = new InMemoryRefreshStore();
    final RefreshTokenService service = new RefreshTokenService(store, clock);

    private Refresh stored(String raw) {
        return store.find(SessionService.hashId(raw)).orElseThrow();
    }

    @Test
    void startThenRotate() {
        String first = service.start(USER);
        var rotated = service.rotate(first);
        assertThat(rotated.userId()).isEqualTo(USER);
        assertThat(rotated.raw()).isNotEqualTo(first);
        assertThat(stored(rotated.raw()).familyId()).isEqualTo(stored(first).familyId());
        assertThat(stored(first).usedAt()).isEqualTo(clock.now);
    }

    @Test
    void storesOnlyTheHash() {
        String raw = service.start(USER);
        assertThat(store.find(raw.getBytes())).isEmpty();
        assertThat(store.find(SessionService.hashId(raw))).isPresent();
    }

    @Test
    void reuseRevokesTheWholeFamily() {
        String first = service.start(USER);
        String second = service.rotate(first).raw();

        assertThatThrownBy(() -> service.rotate(first)).isInstanceOf(RefreshReuseException.class);
        assertThat(store.find(SessionService.hashId(second))).as("legitimate child").isEmpty();
        assertThatThrownBy(() -> service.rotate(second)).isInstanceOf(InvalidRefreshException.class)
                .isNotInstanceOf(RefreshReuseException.class);
    }

    @Test
    void reuseDoesNotTouchOtherFamilies() {
        String a = service.start(USER);
        String b = service.start(USER);
        service.rotate(a);
        assertThatThrownBy(() -> service.rotate(a)).isInstanceOf(RefreshReuseException.class);
        assertThat(service.rotate(b).userId()).isEqualTo(USER);
    }

    @Test
    void rejectsUnknownAndEmpty() {
        assertThatThrownBy(() -> service.rotate("nope")).isInstanceOf(InvalidRefreshException.class);
        assertThatThrownBy(() -> service.rotate("")).isInstanceOf(InvalidRefreshException.class);
        assertThatThrownBy(() -> service.rotate(null)).isInstanceOf(InvalidRefreshException.class);
    }

    @Test
    void idleExpiry() {
        String raw = service.start(USER);
        clock.now = clock.now.plus(RefreshTokenService.TTL).minusSeconds(1);
        String next = service.rotate(raw).raw();
        // the new token has a fresh idle window
        clock.now = clock.now.plus(RefreshTokenService.TTL).minusSeconds(1);
        assertThat(service.rotate(next).userId()).isEqualTo(USER);

        String idle = service.start(USER);
        clock.now = clock.now.plus(RefreshTokenService.TTL);
        assertThatThrownBy(() -> service.rotate(idle)).isInstanceOf(InvalidRefreshException.class)
                .isNotInstanceOf(RefreshReuseException.class);
    }

    /** Rotation never extends a login past its absolute end. */
    @Test
    void familyExpiryCapsEveryToken() {
        Instant start = clock.now;
        String raw = service.start(USER);
        Instant end = start.plus(RefreshTokenService.FAMILY_TTL);

        // Rotate every 6 days: the idle window never lapses, the family still ends at 30 days.
        for (int day = 6; day <= 24; day += 6) {
            clock.now = start.plus(Duration.ofDays(day));
            raw = service.rotate(raw).raw();
            assertThat(stored(raw).expiresAt()).isBeforeOrEqualTo(end);
            assertThat(stored(raw).familyExpiresAt()).isEqualTo(end);
        }
        clock.now = start.plus(Duration.ofDays(29));
        raw = service.rotate(raw).raw();
        assertThat(stored(raw).expiresAt()).as("capped, not 7 days out").isEqualTo(end);

        clock.now = end;
        String last = raw;
        assertThatThrownBy(() -> service.rotate(last)).isInstanceOf(InvalidRefreshException.class);
    }

    @Test
    void revokeEndsTheFamily() {
        String first = service.start(USER);
        String second = service.rotate(first).raw();
        service.revoke(second);
        assertThat(store.find(SessionService.hashId(first))).isEmpty();
        assertThat(store.find(SessionService.hashId(second))).isEmpty();
        service.revoke("unknown"); // no-op
        service.revoke("");
        service.revoke(null);
    }

    @Test
    void purgeRemovesExpired() {
        String raw = service.start(USER);
        assertThat(service.purge()).isZero();
        clock.now = clock.now.plus(RefreshTokenService.TTL).plusSeconds(1);
        assertThat(service.purge()).isEqualTo(1);
        assertThat(store.find(SessionService.hashId(raw))).isEmpty();
    }

    /** A store failure while rotating must leave the old token usable for a retry. */
    @Test
    void failedRotateKeepsTheOldTokenValid() {
        String raw = service.start(USER);
        RefreshStore failing = new RefreshStore() {
            @Override
            public void create(Refresh t) {
                store.create(t);
            }

            @Override
            public java.util.Optional<Refresh> find(byte[] h) {
                return store.find(h);
            }

            @Override
            public boolean rotate(byte[] oldHash, Instant usedAt, Refresh next) {
                throw new IllegalStateException("db down");
            }

            @Override
            public void deleteFamily(UUID f) {
                store.deleteFamily(f);
            }

            @Override
            public long deleteExpired(Instant now) {
                return store.deleteExpired(now);
            }
        };
        assertThatThrownBy(() -> new RefreshTokenService(failing, clock).rotate(raw))
                .isInstanceOf(IllegalStateException.class);
        assertThat(service.rotate(raw).userId()).as("retry after the failure").isEqualTo(USER);
    }
}
