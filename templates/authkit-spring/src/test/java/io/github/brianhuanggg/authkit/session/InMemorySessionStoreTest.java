package io.github.brianhuanggg.authkit.session;

import java.util.UUID;

class InMemorySessionStoreTest implements SessionStoreContract {

    private final SessionStore store = new InMemorySessionStore();

    @Override
    public SessionStore store() {
        return store;
    }

    @Override
    public UUID newUser() {
        return UUID.randomUUID();
    }
}
