package io.github.brianhuanggg.authkit.token;

import java.time.Instant;
import java.time.OffsetDateTime;
import java.time.ZoneOffset;
import java.util.Optional;
import java.util.UUID;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;
import org.springframework.transaction.annotation.Transactional;

/** The {@code refresh_tokens} table, shared with the Go API. */
@Repository
public class PostgresRefreshStore implements RefreshStore {

    private static final String INSERT = """
            INSERT INTO refresh_tokens (id_hash, user_id, family_id, used_at, expires_at, family_expires_at)
            VALUES (:hash, :user, :family, :used, :exp, :familyExp)""";

    private final JdbcClient jdbc;

    public PostgresRefreshStore(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    private static OffsetDateTime utc(Instant i) {
        return i == null ? null : OffsetDateTime.ofInstant(i, ZoneOffset.UTC);
    }

    private int insert(Refresh t) {
        return jdbc.sql(INSERT)
                .param("hash", t.idHash())
                .param("user", t.userId())
                .param("family", t.familyId())
                .param("used", utc(t.usedAt()))
                .param("exp", utc(t.expiresAt()))
                .param("familyExp", utc(t.familyExpiresAt()))
                .update();
    }

    @Override
    public void create(Refresh token) {
        try {
            insert(token);
        } catch (DuplicateKeyException e) {
            throw new DuplicateRefreshException();
        }
    }

    @Override
    public Optional<Refresh> find(byte[] idHash) {
        return jdbc.sql("""
                SELECT id_hash, user_id, family_id, used_at, expires_at, family_expires_at
                FROM refresh_tokens WHERE id_hash = :hash""")
                .param("hash", idHash)
                .query((rs, row) -> {
                    OffsetDateTime used = rs.getObject("used_at", OffsetDateTime.class);
                    return new Refresh(
                            rs.getBytes("id_hash"),
                            rs.getObject("user_id", UUID.class),
                            rs.getObject("family_id", UUID.class),
                            used == null ? null : used.toInstant(),
                            rs.getObject("expires_at", OffsetDateTime.class).toInstant(),
                            rs.getObject("family_expires_at", OffsetDateTime.class).toInstant());
                })
                .optional();
    }

    /**
     * The conditional UPDATE and the INSERT share one transaction. Concurrent
     * callers queue on the row lock, so only one sees an update count of 1; if
     * the INSERT fails, the UPDATE is rolled back with it.
     */
    @Override
    @Transactional
    public boolean rotate(byte[] oldHash, Instant usedAt, Refresh next) {
        int updated = jdbc.sql("UPDATE refresh_tokens SET used_at = :used WHERE id_hash = :hash AND used_at IS NULL")
                .param("used", utc(usedAt))
                .param("hash", oldHash)
                .update();
        if (updated != 1) {
            return false;
        }
        try {
            insert(next);
        } catch (DuplicateKeyException e) {
            throw new DuplicateRefreshException(); // unchecked: rolls the UPDATE back
        }
        return true;
    }

    @Override
    public void deleteFamily(UUID familyId) {
        jdbc.sql("DELETE FROM refresh_tokens WHERE family_id = :family").param("family", familyId).update();
    }

    @Override
    public long deleteExpired(Instant now) {
        return jdbc.sql("DELETE FROM refresh_tokens WHERE expires_at < :now").param("now", utc(now)).update();
    }
}
