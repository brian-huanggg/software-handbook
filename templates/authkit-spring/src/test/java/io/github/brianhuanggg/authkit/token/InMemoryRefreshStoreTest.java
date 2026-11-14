package io.github.brianhuanggg.authkit.token;

import static org.assertj.core.api.Assertions.assertThat;

import java.time.Instant;
import java.util.Optional;
import java.util.UUID;
import org.junit.jupiter.api.Test;

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

    @Test
    void rotateUsesTheRoleLookupAndRefusesMissingUsers() {
        UUID admin = UUID.randomUUID();
        UUID gone = UUID.randomUUID();
        var s = new InMemoryRefreshStore(id -> id.equals(admin) ? Optional.of("admin") : Optional.empty());
        Instant exp = Instant.now().plusSeconds(60);
        UUID family = UUID.randomUUID();
        s.create(new Refresh(new byte[] {1}, admin, family, null, exp, exp));
        s.create(new Refresh(new byte[] {2}, gone, family, null, exp, exp));

        assertThat(s.rotate(new byte[] {1}, Instant.now(), new Refresh(new byte[] {3}, admin, family, null, exp, exp)))
                .contains("admin");
        assertThat(s.rotate(new byte[] {2}, Instant.now(), new Refresh(new byte[] {4}, gone, family, null, exp, exp)))
                .as("missing user").isEmpty();
        assertThat(s.find(new byte[] {2}).orElseThrow().usedAt()).as("nothing consumed").isNull();
        assertThat(s.find(new byte[] {4})).isEmpty();
    }
}
