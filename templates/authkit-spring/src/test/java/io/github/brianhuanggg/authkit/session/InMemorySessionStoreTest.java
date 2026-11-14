package io.github.brianhuanggg.authkit.session;

import static org.assertj.core.api.Assertions.assertThat;

import java.time.Instant;
import java.util.Optional;
import java.util.UUID;
import org.junit.jupiter.api.Test;

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

    @Test
    void findWithRoleUsesTheLookup() {
        UUID admin = UUID.randomUUID();
        UUID gone = UUID.randomUUID();
        SessionStore s = new InMemorySessionStore(id -> id.equals(admin) ? Optional.of("admin") : Optional.empty());
        Session a = new Session(new byte[] {1}, admin, Instant.now().plusSeconds(60));
        Session b = new Session(new byte[] {2}, gone, Instant.now().plusSeconds(60));
        s.create(a);
        s.create(b);
        assertThat(s.findWithRole(a.idHash())).get().extracting(SessionUser::role).isEqualTo("admin");
        assertThat(s.findWithRole(b.idHash())).as("missing user").isEmpty();
    }
}
