package io.github.brianhuanggg.authkit.user;

import java.sql.ResultSet;
import java.sql.SQLException;
import java.time.OffsetDateTime;
import java.util.Optional;
import java.util.UUID;
import org.springframework.dao.DataIntegrityViolationException;
import org.springframework.dao.DuplicateKeyException;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.stereotype.Repository;

@Repository
public class UserRepository {

    private static final String COLS = "id, email::text AS email, password_hash, role, created_at";
    private static final String EMAIL_FORMAT_CHECK = "users_email_format";

    private final JdbcClient jdbc;

    public UserRepository(JdbcClient jdbc) {
        this.jdbc = jdbc;
    }

    /**
     * Inserts a user with the default role.
     *
     * @throws EmailTakenException   the email exists (case-insensitive)
     * @throws InvalidEmailException the email fails the format check
     */
    public User create(String email, String passwordHash) {
        try {
            return jdbc.sql("INSERT INTO users (email, password_hash) VALUES (:email, :hash) RETURNING " + COLS)
                    .param("email", email)
                    .param("hash", passwordHash)
                    .query(UserRepository::map)
                    .single();
        } catch (DuplicateKeyException e) {
            throw new EmailTakenException();
        } catch (DataIntegrityViolationException e) {
            if (e.getMessage() != null && e.getMessage().contains(EMAIL_FORMAT_CHECK)) {
                throw new InvalidEmailException();
            }
            throw e;
        }
    }

    public Optional<User> findByEmail(String email) {
        return jdbc.sql("SELECT " + COLS + " FROM users WHERE email = CAST(:email AS citext)")
                .param("email", email)
                .query(UserRepository::map)
                .optional();
    }

    public Optional<User> findById(UUID id) {
        return jdbc.sql("SELECT " + COLS + " FROM users WHERE id = :id")
                .param("id", id)
                .query(UserRepository::map)
                .optional();
    }

    /**
     * Sets a new hash only if the stored one is still {@code oldHash}, so an
     * upgrade racing with a password change cannot undo it.
     *
     * @return whether the row was updated
     */
    public boolean replacePasswordHash(UUID id, String oldHash, String newHash) {
        return jdbc.sql("UPDATE users SET password_hash = :new WHERE id = :id AND password_hash = :old")
                .param("id", id)
                .param("old", oldHash)
                .param("new", newHash)
                .update() == 1;
    }

    private static User map(ResultSet rs, int row) throws SQLException {
        return new User(
                rs.getObject("id", UUID.class),
                rs.getString("email"),
                rs.getString("password_hash"),
                rs.getString("role"),
                rs.getObject("created_at", OffsetDateTime.class).toInstant());
    }
}
