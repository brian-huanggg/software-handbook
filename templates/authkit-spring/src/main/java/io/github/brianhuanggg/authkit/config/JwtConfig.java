package io.github.brianhuanggg.authkit.config;

import io.github.brianhuanggg.authkit.token.JwtService;
import java.time.Clock;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.oauth2.jwt.JwtDecoder;

@Configuration(proxyBeanMethods = false)
class JwtConfig {

    @Bean
    JwtService jwtService(AuthkitProperties props, Clock clock) {
        return new JwtService(props.jwtSecret(), props.jwtPreviousSecrets(), props.jwtIssuer(), clock);
    }

    /** Defined here so Boot does not build its own decoder from properties. */
    @Bean
    JwtDecoder jwtDecoder(JwtService jwt) {
        return jwt.decoder();
    }
}
