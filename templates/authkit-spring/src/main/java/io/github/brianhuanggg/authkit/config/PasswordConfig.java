package io.github.brianhuanggg.authkit.config;

import io.github.brianhuanggg.authkit.password.PepperedArgon2Encoder;
import java.nio.charset.StandardCharsets;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.crypto.password.PasswordEncoder;

@Configuration(proxyBeanMethods = false)
class PasswordConfig {

    /** One slot per CPU: more only adds memory pressure (19 MiB per hash), not throughput. */
    @Bean
    PasswordEncoder passwordEncoder(AuthkitProperties props) {
        byte[] pepper = props.passwordPepper() == null
                ? null : props.passwordPepper().getBytes(StandardCharsets.UTF_8);
        var previous = props.passwordPreviousPeppers().stream()
                .map(p -> p.getBytes(StandardCharsets.UTF_8)).toList();
        return new PepperedArgon2Encoder(pepper, previous, Runtime.getRuntime().availableProcessors());
    }
}
