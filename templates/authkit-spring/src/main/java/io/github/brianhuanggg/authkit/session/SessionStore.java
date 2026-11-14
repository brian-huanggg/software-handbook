package io.github.brianhuanggg.authkit.session;

import java.time.Instant;
import java.util.Optional;

/**
 * Persists sessions by ID hash. Implementations do not enforce expiry:
 * {@link #find} returns expired rows and {@link SessionService} decides.
 */
public interface SessionStore {

    /** @throws DuplicateSessionException if the hash already exists */
    void create(Session session);

    Optional<Session> find(byte[] idHash);

    /**
     * Like {@link #find}, plus the user's current role, in one lookup. Empty
     * when the session or its user is missing. Expired rows are returned.
     */
    Optional<SessionUser> findWithRole(byte[] idHash);

    /** Idempotent: deleting a missing session is not an error. */
    void delete(byte[] idHash);

    /** Removes sessions that expired before {@code now}; returns how many. */
    long deleteExpired(Instant now);
}
