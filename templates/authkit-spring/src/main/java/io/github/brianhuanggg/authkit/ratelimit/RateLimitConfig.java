package io.github.brianhuanggg.authkit.ratelimit;

import io.github.brianhuanggg.authkit.config.AuthkitProperties;
import java.time.Duration;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration(proxyBeanMethods = false)
class RateLimitConfig {

    /** Burst of 20 per IP, then one request every 3 seconds. */
    static final Duration IP_EVERY = Duration.ofSeconds(3);
    /** Burst of 5 per email, then one attempt every 30 seconds. */
    static final Duration EMAIL_EVERY = Duration.ofSeconds(30);

    @Bean
    RateLimits rateLimits(AuthkitProperties props) {
        return new RateLimits(
                new Limiter(IP_EVERY, props.rateLimitIpBurst()),
                new Limiter(EMAIL_EVERY, props.rateLimitEmailBurst()));
    }
}
