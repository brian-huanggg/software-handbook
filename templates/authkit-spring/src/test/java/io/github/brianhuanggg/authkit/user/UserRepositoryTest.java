package io.github.brianhuanggg.authkit.user;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

class UserRepositoryTest extends PostgresTestBase {

    @Autowired
    UserRepository users;

    private static String uniqueEmail() {
        return "u-" + UUID.randomUUID() + "@example.com";
    }

    @Test
    void createThenLookup() {
        String email = uniqueEmail();
        User u = users.create(email, "hash");
        assertThat(u.id()).isNotNull();
        assertThat(u.email()).isEqualTo(email);
        assertThat(u.passwordHash()).isEqualTo("hash");
        assertThat(u.role()).isEqualTo("user");
        assertThat(u.createdAt()).isNotNull();

        assertThat(users.findByEmail(email)).get().extracting(User::id).isEqualTo(u.id());
        assertThat(users.findById(u.id())).get().extracting(User::email).isEqualTo(email);
    }

    @Test
    void duplicateEmailIsCaseInsensitive() {
        String email = uniqueEmail();
        users.create(email, "h");
        assertThatThrownBy(() -> users.create(email, "h")).isInstanceOf(EmailTakenException.class);
        assertThatThrownBy(() -> users.create(email.toUpperCase(), "h"))
                .isInstanceOf(EmailTakenException.class);
    }

    @Test
    void findByEmailIsCaseInsensitive() {
        String email = uniqueEmail();
        User u = users.create(email, "h");
        assertThat(users.findByEmail(email.toUpperCase())).get().extracting(User::id).isEqualTo(u.id());
    }

    @Test
    void notFound() {
        assertThat(users.findByEmail(uniqueEmail())).isEmpty();
        assertThat(users.findById(UUID.randomUUID())).isEmpty();
    }

    @Test
    void invalidEmail() {
        assertThatThrownBy(() -> users.create("not-an-email", "h")).isInstanceOf(InvalidEmailException.class);
    }

    @Test
    void replacePasswordHashIsConditional() {
        User u = users.create(uniqueEmail(), "old");
        assertThat(users.replacePasswordHash(u.id(), "stale", "new")).isFalse();
        assertThat(users.replacePasswordHash(u.id(), "old", "new")).isTrue();
        assertThat(users.findById(u.id())).get().extracting(User::passwordHash).isEqualTo("new");
    }
}
