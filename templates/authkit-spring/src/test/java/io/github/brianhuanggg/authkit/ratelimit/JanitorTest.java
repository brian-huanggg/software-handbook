package io.github.brianhuanggg.authkit.ratelimit;

import static org.assertj.core.api.Assertions.assertThat;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import io.github.brianhuanggg.authkit.session.Session;
import io.github.brianhuanggg.authkit.session.SessionService;
import io.github.brianhuanggg.authkit.session.SessionStore;
import io.github.brianhuanggg.authkit.token.Refresh;
import io.github.brianhuanggg.authkit.token.RefreshStore;
import io.github.brianhuanggg.authkit.user.UserRepository;
import java.security.SecureRandom;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;

class JanitorTest extends PostgresTestBase {

    @Autowired
    Janitor janitor;
    @Autowired
    SessionStore sessions;
    @Autowired
    RefreshStore refresh;
    @Autowired
    UserRepository users;

    private static byte[] hash() {
        byte[] b = new byte[32];
        new SecureRandom().nextBytes(b);
        return b;
    }

    @Test
    void purgeDeletesExpiredSessionsAndRefreshTokensOnly() {
        UUID user = users.create("j-" + UUID.randomUUID() + "@example.com", "hash").id();
        Instant past = Instant.now().minus(1, ChronoUnit.HOURS);
        Instant future = Instant.now().plus(1, ChronoUnit.HOURS);

        Session deadSession = new Session(hash(), user, past);
        Session liveSession = new Session(hash(), user, future);
        sessions.create(deadSession);
        sessions.create(liveSession);
        Refresh deadToken = new Refresh(hash(), user, UUID.randomUUID(), null, past, future);
        Refresh liveToken = new Refresh(hash(), user, UUID.randomUUID(), null, future, future);
        refresh.create(deadToken);
        refresh.create(liveToken);

        janitor.purge();

        assertThat(sessions.find(deadSession.idHash())).isEmpty();
        assertThat(sessions.find(liveSession.idHash())).isPresent();
        assertThat(refresh.find(deadToken.idHash())).isEmpty();
        assertThat(refresh.find(liveToken.idHash())).isPresent();
    }
}
