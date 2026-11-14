package io.github.brianhuanggg.authkit.ratelimit;

import io.github.brianhuanggg.authkit.session.SessionStore;
import io.github.brianhuanggg.authkit.token.RefreshTokenService;
import java.time.Clock;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.scheduling.annotation.EnableScheduling;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

/**
 * Deletes expired sessions and refresh tokens every 10 minutes. Expired rows
 * are already rejected on use, so this only keeps the tables small.
 */
@Component
@EnableScheduling
public class Janitor {

    private static final Logger log = LoggerFactory.getLogger(Janitor.class);
    private static final long INTERVAL_MS = 10 * 60 * 1000L;

    private final SessionStore sessions;
    private final RefreshTokenService refresh;
    private final Clock clock;

    Janitor(SessionStore sessions, RefreshTokenService refresh, Clock clock) {
        this.sessions = sessions;
        this.refresh = refresh;
        this.clock = clock;
    }

    // The first run waits a full interval: nothing needs purging at boot.
    @Scheduled(fixedDelay = INTERVAL_MS, initialDelay = INTERVAL_MS)
    public void purge() {
        // Independent: a failure in one must not skip the other, or stop the schedule.
        try {
            long n = sessions.deleteExpired(clock.instant());
            if (n > 0) {
                log.info("purged expired rows table=sessions count={}", n);
            }
        } catch (RuntimeException e) {
            log.error("purge sessions", e);
        }
        try {
            long n = refresh.purge();
            if (n > 0) {
                log.info("purged expired rows table=refresh_tokens count={}", n);
            }
        } catch (RuntimeException e) {
            log.error("purge refresh_tokens", e);
        }
    }
}
