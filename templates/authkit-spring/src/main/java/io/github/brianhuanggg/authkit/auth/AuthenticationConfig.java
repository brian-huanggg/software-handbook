package io.github.brianhuanggg.authkit.auth;

import io.github.brianhuanggg.authkit.user.User;
import io.github.brianhuanggg.authkit.user.UserRepository;
import java.time.Clock;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.ProviderManager;
import org.springframework.security.authentication.dao.DaoAuthenticationProvider;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.security.core.userdetails.UserDetailsPasswordService;
import org.springframework.security.core.userdetails.UserDetailsService;
import org.springframework.security.core.userdetails.UsernameNotFoundException;
import org.springframework.security.crypto.password.PasswordEncoder;

@Configuration(proxyBeanMethods = false)
class AuthenticationConfig {

    private static final Logger log = LoggerFactory.getLogger(AuthenticationConfig.class);

    @Bean
    Clock clock() {
        return Clock.systemUTC();
    }

    /**
     * Email and password login. {@link DaoAuthenticationProvider} gives two things
     * the Go API wrote by hand: an unknown email still costs one password check
     * (so timing does not reveal accounts), and a hash made with old parameters
     * or a previous pepper is upgraded after a successful login.
     *
     * <p>Neither the {@code UserDetailsService} nor the password service is a
     * bean, so Spring Security cannot wire them into other chains.
     */
    @Bean
    AuthenticationManager passwordAuthenticationManager(UserRepository users, PasswordEncoder encoder) {
        UserDetailsService byEmail = email -> users.findByEmail(email)
                .map(AuthenticationConfig::details)
                .orElseThrow(() -> new UsernameNotFoundException("no such user"));

        // Conditional update: an upgrade racing with a password change cannot undo it.
        // Failure is logged, not fatal: the login itself already succeeded.
        UserDetailsPasswordService upgrade = (user, newHash) -> {
            AccountUserDetails account = (AccountUserDetails) user;
            try {
                users.replacePasswordHash(account.id(), account.passwordHash(), newHash);
            } catch (RuntimeException e) {
                log.warn("upgrade password hash", e);
            }
            return new AccountUserDetails(account.id(), account.email(), newHash, account.role());
        };

        DaoAuthenticationProvider provider = new DaoAuthenticationProvider(byEmail);
        provider.setPasswordEncoder(encoder);
        provider.setUserDetailsPasswordService(upgrade);
        return new ProviderManager(provider);
    }

    private static UserDetails details(User u) {
        return new AccountUserDetails(u.id(), u.email(), u.passwordHash(), u.role());
    }
}
