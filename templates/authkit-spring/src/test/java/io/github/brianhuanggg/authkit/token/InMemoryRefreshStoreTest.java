package io.github.brianhuanggg.authkit.token;

import java.util.UUID;

class InMemoryRefreshStoreTest implements RefreshStoreContract {

    private final RefreshStore store = new InMemoryRefreshStore();

    @Override
    public RefreshStore store() {
        return store;
    }

    @Override
    public UUID newUser() {
        return UUID.randomUUID();
    }
}
