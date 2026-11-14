package io.github.brianhuanggg.authkit.auth;

import io.github.brianhuanggg.authkit.web.ApiException;
import io.github.brianhuanggg.authkit.web.Credentials;
import java.nio.charset.StandardCharsets;
import org.springframework.http.HttpStatus;
import org.springframework.security.authentication.AuthenticationManager;
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken;
import org.springframework.security.core.AuthenticationException;
import org.springframework.stereotype.Component;

/** Credential check shared by {@code /session/login} and {@code /token/login}. */
@Component
public class LoginAuthenticator {

    /** The longest password {@code /register} accepts; no stored one can be longer. */
    public static final int MAX_PASSWORD_BYTES = 128;

    private final AuthenticationManager manager;

    LoginAuthenticator(AuthenticationManager passwordAuthenticationManager) {
        this.manager = passwordAuthenticationManager;
    }

    /**
     * @throws ApiException 400 for missing fields, 401 for any bad credentials
     *         (one message for an unknown account and a wrong password)
     */
    public AccountUserDetails authenticate(Credentials c) {
        if (c.email() == null || c.email().isEmpty() || c.password() == null || c.password().isEmpty()) {
            throw new ApiException(HttpStatus.BAD_REQUEST, "email and password are required");
        }
        // An over-long password is simply wrong; refuse it before paying for a hash.
        if (c.password().getBytes(StandardCharsets.UTF_8).length > MAX_PASSWORD_BYTES) {
            throw invalid();
        }
        try {
            var result = manager.authenticate(UsernamePasswordAuthenticationToken.unauthenticated(
                    c.email(), c.password()));
            return (AccountUserDetails) result.getPrincipal();
        } catch (AuthenticationException e) {
            throw invalid();
        }
    }

    private static ApiException invalid() {
        return new ApiException(HttpStatus.UNAUTHORIZED, "invalid email or password");
    }
}
