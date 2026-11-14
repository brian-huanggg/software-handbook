package io.github.brianhuanggg.authkit.session;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import io.github.brianhuanggg.authkit.user.UserRepository;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Autowired;

class PostgresSessionStoreTest extends PostgresTestBase implements SessionStoreContract {

    @Autowired
    PostgresSessionStore store;

    @Autowired
    UserRepository users;

    @Override
    public SessionStore store() {
        return store;
    }

    @Override
    public UUID newUser() {
        return users.create("u-" + UUID.randomUUID() + "@example.com", "hash").id();
    }
}
