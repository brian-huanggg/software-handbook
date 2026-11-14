package io.github.brianhuanggg.authkit.session;

import java.time.Instant;
import java.util.HexFormat;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import java.util.function.Function;

/** For tests and single-process demos; sessions vanish on restart. Not a bean. */
public final class InMemorySessionStore implements SessionStore {

    private final Map<String, Session> sessions = new ConcurrentHashMap<>();
    private final Function<UUID, Optional<String>> roleOf;

    /** Every user exists and has the role {@code user}. */
    public InMemorySessionStore() {
        this(id -> Optional.of("user"));
    }

    /**
     * @param roleOf the current role of a user, empty when the user does not
     *               exist. There is no users table here, so the caller supplies it.
     */
    public InMemorySessionStore(Function<UUID, Optional<String>> roleOf) {
        this.roleOf = roleOf;
    }

    private static String key(byte[] idHash) {
        return HexFormat.of().formatHex(idHash);
    }

    @Override
    public void create(Session session) {
        // The record keeps the caller's array, so copy: a later mutation must not change the key.
        Session copy = new Session(session.idHash().clone(), session.userId(), session.expiresAt());
        if (sessions.putIfAbsent(key(copy.idHash()), copy) != null) {
            throw new DuplicateSessionException();
        }
    }

    @Override
    public Optional<Session> find(byte[] idHash) {
        Session s = sessions.get(key(idHash));
        return s == null ? Optional.empty()
                : Optional.of(new Session(s.idHash().clone(), s.userId(), s.expiresAt()));
    }

    @Override
    public Optional<SessionUser> findWithRole(byte[] idHash) {
        return find(idHash).flatMap(s -> roleOf.apply(s.userId()).map(role -> new SessionUser(s, role)));
    }

    @Override
    public void delete(byte[] idHash) {
        sessions.remove(key(idHash));
    }

    @Override
    public long deleteExpired(Instant now) {
        long before = sessions.size();
        sessions.values().removeIf(s -> s.expiresAt().isBefore(now));
        return before - sessions.size();
    }
}
