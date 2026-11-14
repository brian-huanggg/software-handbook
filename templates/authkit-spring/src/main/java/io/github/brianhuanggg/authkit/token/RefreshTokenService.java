package io.github.brianhuanggg.authkit.token;

import io.github.brianhuanggg.authkit.session.SessionService;
import java.security.SecureRandom;
import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.util.Base64;
import java.util.Optional;
import java.util.UUID;
import org.springframework.stereotype.Service;

/**
 * Issues, rotates and revokes opaque refresh tokens. The raw value exists
 * only in the client's cookie; the store sees its SHA-256.
 */
@Service
public class RefreshTokenService {

    /** Idle lifetime of each token. */
    public static final Duration TTL = Duration.ofDays(7);
    /** Absolute lifetime of a login chain. */
    public static final Duration FAMILY_TTL = Duration.ofDays(30);

    private static final SecureRandom RANDOM = new SecureRandom();

    private final RefreshStore store;
    private final Clock clock;

    public RefreshTokenService(RefreshStore store, Clock clock) {
        this.store = store;
        this.clock = clock;
    }

    /** A new token and the user it belongs to. */
    public record Rotated(String raw, UUID userId) {
    }

    private record Fresh(String raw, Refresh token) {
    }

    private Fresh next(UUID userId, UUID familyId, Instant familyExpires) {
        byte[] random = new byte[32];
        RANDOM.nextBytes(random);
        String raw = Base64.getUrlEncoder().withoutPadding().encodeToString(random);
        Instant exp = clock.instant().plus(TTL);
        if (exp.isAfter(familyExpires)) {
            exp = familyExpires;
        }
        return new Fresh(raw, new Refresh(SessionService.hashId(raw), userId, familyId, null, exp, familyExpires));
    }

    /** Begins a new family (a login) and returns its first raw token. */
    public String start(UUID userId) {
        Fresh f = next(userId, UUID.randomUUID(), clock.instant().plus(FAMILY_TTL));
        store.create(f.token());
        return f.raw();
    }

    /**
     * Consumes {@code raw} and returns its replacement plus the owning user.
     *
     * <p>Presenting a token that was already rotated is treated as theft: the
     * whole family is revoked. Losing a race between two concurrent uses of
     * the same token is handled the same way, so a client that double-submits
     * is logged out and must sign in again.
     *
     * <p>Consuming the old token and storing its replacement is one atomic
     * store step: a failure leaves the old token unused, so the client can
     * retry without tripping reuse detection.
     *
     * @throws InvalidRefreshException unknown, expired or revoked
     * @throws RefreshReuseException reuse detected, family revoked
     */
    public Rotated rotate(String raw) {
        if (raw == null || raw.isEmpty()) {
            throw new InvalidRefreshException();
        }
        byte[] hash = SessionService.hashId(raw);
        Refresh cur = store.find(hash).orElseThrow(InvalidRefreshException::new);
        Instant now = clock.instant();
        if (!now.isBefore(cur.expiresAt()) || !now.isBefore(cur.familyExpiresAt())) {
            throw new InvalidRefreshException();
        }
        if (cur.usedAt() != null) {
            throw revokeForReuse(cur);
        }
        Fresh next = next(cur.userId(), cur.familyId(), cur.familyExpiresAt());
        if (!store.rotate(hash, now, next.token())) {
            throw revokeForReuse(cur);
        }
        return new Rotated(next.raw(), cur.userId());
    }

    private RefreshReuseException revokeForReuse(Refresh cur) {
        store.deleteFamily(cur.familyId());
        return new RefreshReuseException(cur.familyId(), cur.userId());
    }

    /** Ends the family {@code raw} belongs to (logout). Unknown tokens are a no-op. */
    public void revoke(String raw) {
        if (raw == null || raw.isEmpty()) {
            return;
        }
        Optional<Refresh> cur = store.find(SessionService.hashId(raw));
        cur.ifPresent(t -> store.deleteFamily(t.familyId()));
    }

    /** Deletes expired tokens. Run it periodically. */
    public long purge() {
        return store.deleteExpired(clock.instant());
    }
}
