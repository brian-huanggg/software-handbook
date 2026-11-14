package io.github.brianhuanggg.authkit.session;

import java.time.Instant;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.util.Optional;
import java.util.UUID;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

/** The {@code sessions} table, shared with the Go API. */
@Repository
public class PostgresSessionStore implements SessionStore {

    private final JdbcClient jdbc;

    public PostgresSessionStore(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    @Override
    public void create(Session session) {
        try {
            jdbc.sql("INSERT INTO sessions (id_hash, user_id, expires_at) VALUES (:hash, :user, :exp)")
                    .param("hash", session.idHash())
                    .param("user", session.userId())
                    .param("exp", OffsetDateTime.ofInstant(session.expiresAt(), ZoneOffset.UTC))
                    .update();
        } catch (DuplicateKeyException e) {
            throw new DuplicateSessionException();
        }
    }

    @Override
    public Optional<Session> find(byte[] idHash) {
        return jdbc.sql("SELECT id_hash, user_id, expires_at FROM sessions WHERE id_hash = :hash")
                .param("hash", idHash)
                .query((rs, row) -> new Session(
                        rs.getBytes("id_hash"),
                        rs.getObject("user_id", UUID.class),
                        rs.getObject("expires_at", OffsetDateTime.class).toInstant()))
                .optional();
    }

    /**
     * One round trip for every authenticated request. The foreign key cascades
     * on user deletion, so the inner join loses no live session.
     */
    @Override
    public Optional<SessionUser> findWithRole(byte[] idHash) {
        return jdbc.sql("""
                SELECT s.id_hash, s.user_id, s.expires_at, u.role
                FROM sessions s JOIN users u ON u.id = s.user_id
                WHERE s.id_hash = :hash""")
                .param("hash", idHash)
                .query((rs, row) -> new SessionUser(
                        new Session(
                                rs.getBytes("id_hash"),
                                rs.getObject("user_id", UUID.class),
                                rs.getObject("expires_at", OffsetDateTime.class).toInstant()),
                        rs.getString("role")))
                .optional();
    }

    @Override
    public void delete(byte[] idHash) {
        jdbc.sql("DELETE FROM sessions WHERE id_hash = :hash").param("hash", idHash).update();
    }

    @Override
    public long deleteExpired(Instant now) {
        return jdbc.sql("DELETE FROM sessions WHERE expires_at < :now")
                .param("now", OffsetDateTime.ofInstant(now, ZoneOffset.UTC))
                .update();
    }
}
