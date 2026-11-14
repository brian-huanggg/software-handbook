package io.github.brianhuanggg.authkit.session;

import static org.assertj.core.api.Assertions.assertThat;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import io.github.brianhuanggg.authkit.user.UserRepository;
import java.time.Instant;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.simple.JdbcClient;

class PostgresSessionStoreTest extends PostgresTestBase implements SessionStoreContract {

    @Autowired
    PostgresSessionStore store;

    @Autowired
    UserRepository users;

    @Autowired
    JdbcClient jdbc;

    @Override
    public SessionStore store() {
        return store;
    }

    @Override
    public UUID newUser() {
        return users.create("u-" + UUID.randomUUID() + "@example.com", "hash").id();
    }

    @Test
    void findWithRoleReadsTheCurrentRoleAndDropsDeletedUsers() {
        UUID user = newUser();
        Session s = new Session(new byte[] {7, 7, 7}, user, Instant.now().plusSeconds(60));
        store.create(s);

        jdbc.sql("UPDATE users SET role = 'admin' WHERE id = :id").param("id", user).update();
        assertThat(store.findWithRole(s.idHash())).get().extracting(SessionUser::role).isEqualTo("admin");

        jdbc.sql("DELETE FROM users WHERE id = :id").param("id", user).update();
        assertThat(store.findWithRole(s.idHash())).isEmpty();
    }
}
