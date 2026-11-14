package io.github.brianhuanggg.authkit.user;

import java.time.Instant;
import java.util.Locale;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;
import java.util.concurrent.ConcurrentHashMap;
import java.util.regex.Pattern;
import org.springframework.dao.DataAccessResourceFailureException;

/**
 * A {@link UserRepository} without a database, for tests that need users but
 * not Postgres. It mirrors the table's rules: case-insensitive unique email,
 * the {@code users_email_format} check, and the default role. {@link #down}
 * makes every call fail as if the database were unreachable.
 */
public class InMemoryUserRepository extends UserRepository {

    // Same expression as the users_email_format CHECK in V1__init_auth.sql.
    private static final Pattern EMAIL = Pattern.compile(
            "^[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(\\.[A-Za-z0-9-]+)*\\.[A-Za-z]{2,}$");

    private final Map<String, User> byEmail = new ConcurrentHashMap<>();
    private volatile boolean down;

    public InMemoryUserRepository() {
        super(null); // every method is overridden, so the JdbcClient is never used
    }

    /** While true, every call throws like a JDBC failure does. */
    public void down(boolean down) {
        this.down = down;
    }

    private void check() {
        if (down) {
            throw new DataAccessResourceFailureException("database is down (test)");
        }
    }

    private static String key(String email) {
        return email.toLowerCase(Locale.ROOT);
    }

    @Override
    public User create(String email, String passwordHash) {
        check();
        if (email.length() > 254 || !EMAIL.matcher(email).matches()) {
            throw new InvalidEmailException();
        }
        User u = new User(UUID.randomUUID(), email, passwordHash, "user", Instant.now());
        if (byEmail.putIfAbsent(key(email), u) != null) {
            throw new EmailTakenException();
        }
        return u;
    }

    @Override
    public Optional<User> findByEmail(String email) {
        check();
        return Optional.ofNullable(byEmail.get(key(email)));
    }

    @Override
    public Optional<User> findById(UUID id) {
        check();
        return byEmail.values().stream().filter(u -> u.id().equals(id)).findFirst();
    }

    @Override
    public boolean replacePasswordHash(UUID id, String oldHash, String newHash) {
        check();
        for (Map.Entry<String, User> e : byEmail.entrySet()) {
            User u = e.getValue();
            if (u.id().equals(id) && u.passwordHash().equals(oldHash)) {
                return byEmail.replace(e.getKey(), u,
                        new User(u.id(), u.email(), newHash, u.role(), u.createdAt()));
            }
        }
        return false;
    }
}
