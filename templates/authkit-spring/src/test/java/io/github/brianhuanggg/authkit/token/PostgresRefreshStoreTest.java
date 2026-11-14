package io.github.brianhuanggg.authkit.token;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import io.github.brianhuanggg.authkit.user.UserRepository;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Autowired;

class PostgresRefreshStoreTest extends PostgresTestBase implements RefreshStoreContract {

    @Autowired
    PostgresRefreshStore store;

    @Autowired
    UserRepository users;

    @Override
    public RefreshStore store() {
        return store;
    }

    @Override
    public UUID newUser() {
        return users.create("u-" + UUID.randomUUID() + "@example.com", "hash").id();
    }
}
