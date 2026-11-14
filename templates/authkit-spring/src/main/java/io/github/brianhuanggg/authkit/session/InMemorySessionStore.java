package io.github.brianhuanggg.authkit.session;

import java.time.Instant;
import java.util.HexFormat;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;

/** For tests and single-process demos; sessions vanish on restart. Not a bean. */
public final class InMemorySessionStore implements SessionStore {

    private final Map<String, Session> sessions = new ConcurrentHashMap<>();

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
