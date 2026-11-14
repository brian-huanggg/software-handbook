package io.github.brianhuanggg.authkit.token;

import java.time.Instant;
import java.util.Optional;
import java.util.UUID;

/** Persists refresh tokens by ID hash. Expiry is decided by {@link RefreshTokenService}, not here. */
public interface RefreshStore {

    /** @throws DuplicateRefreshException if the hash already exists */
    void create(Refresh token);

    Optional<Refresh> find(byte[] idHash);

    /**
     * Atomically marks {@code oldHash} used (only if it exists, is still
     * unused and its user exists) and stores {@code next}, all or nothing.
     * When this call did so, returns the user's current role, read in the
     * same step, so the caller needs no further lookup after the rotation has
     * committed. Empty means this call did not rotate, and nothing changed;
     * on an exception nothing changed either. Rotation relies on exactly one
     * concurrent caller winning.
     *
     * @throws DuplicateRefreshException if {@code next} collides; the old token stays unused
     */
    Optional<String> rotate(byte[] oldHash, Instant usedAt, Refresh next);

    /** Removes every token of the family. An unknown family is not an error. */
    void deleteFamily(UUID familyId);

    /**
     * Removes tokens whose {@code expiresAt} is before {@code now}; returns
     * how many. {@code expiresAt} never passes the family's end, so one check
     * covers both. Expired tokens are rejected before reuse detection runs,
     * so purging them loses nothing.
     */
    long deleteExpired(Instant now);
}
