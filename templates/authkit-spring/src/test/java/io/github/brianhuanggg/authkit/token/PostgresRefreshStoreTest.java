package io.github.brianhuanggg.authkit.token;

import static org.assertj.core.api.Assertions.assertThat;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import io.github.brianhuanggg.authkit.crypto.OpaqueIds;
import io.github.brianhuanggg.authkit.user.UserRepository;
import java.time.Instant;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.jdbc.core.simple.JdbcClient;

class PostgresRefreshStoreTest extends PostgresTestBase implements RefreshStoreContract {

    @Autowired
    PostgresRefreshStore store;

    @Autowired
    UserRepository users;

    @Autowired
    JdbcClient jdbc;

    private static Refresh token(UUID user, UUID family) {
        Instant exp = Instant.now().plusSeconds(3600);
        return new Refresh(OpaqueIds.hash(OpaqueIds.generate()), user, family, null, exp, exp);
    }

    @Override
    public RefreshStore store() {
        return store;
    }

    @Override
    public UUID newUser() {
        return users.create("u-" + UUID.randomUUID() + "@example.com", "hash").id();
    }

    @Test
    void rotateReturnsTheCurrentRole() {
        UUID user = newUser();
        UUID family = UUID.randomUUID();
        Refresh old = token(user, family);
        store.create(old);
        jdbc.sql("UPDATE users SET role = 'admin' WHERE id = :id").param("id", user).update();
        assertThat(store.rotate(old.idHash(), Instant.now(), token(user, family))).contains("admin");
    }
}
