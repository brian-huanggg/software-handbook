package io.github.brianhuanggg.authkit.web;

import io.github.brianhuanggg.authkit.auth.AccountUserDetails;
import io.github.brianhuanggg.authkit.auth.AuthenticatedPrincipal;
import io.github.brianhuanggg.authkit.auth.LoginAuthenticator;
import io.github.brianhuanggg.authkit.config.AuthkitProperties;
import io.github.brianhuanggg.authkit.token.InvalidRefreshException;
import io.github.brianhuanggg.authkit.token.JwtService;
import io.github.brianhuanggg.authkit.token.RefreshReuseException;
import io.github.brianhuanggg.authkit.token.RefreshTokenService;
import io.github.brianhuanggg.authkit.user.User;
import io.github.brianhuanggg.authkit.user.UserRepository;
import java.time.Duration;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseCookie;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.CookieValue;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

/**
 * Token mode: a short-lived access JWT in the body and a rotating refresh
 * token in an HttpOnly cookie that is sent only to {@code /token/*}.
 */
@RestController
class TokenController {

    static final String COOKIE_NAME = "refresh_token";
    private static final String COOKIE_PATH = "/token";
    private static final Logger log = LoggerFactory.getLogger(TokenController.class);

    private final LoginAuthenticator logins;
    private final RefreshTokenService refresh;
    private final JwtService jwt;
    private final UserRepository users;
    private final boolean secure;

    TokenController(LoginAuthenticator logins, RefreshTokenService refresh, JwtService jwt, UserRepository users,
            AuthkitProperties props) {
        this.logins = logins;
        this.refresh = refresh;
        this.jwt = jwt;
        this.users = users;
        this.secure = !props.insecureCookies();
    }

    @PostMapping("/token/login")
    ResponseEntity<Map<String, Object>> login(@RequestBody Credentials c) {
        AccountUserDetails user = logins.authenticate(c);
        String raw = refresh.start(user.id());
        return access(raw, user.id(), user.role());
    }

    @PostMapping("/token/refresh")
    ResponseEntity<Map<String, Object>> refresh(@CookieValue(name = COOKIE_NAME, required = false) String raw) {
        if (raw == null || raw.isEmpty()) {
            return unauthenticated(false);
        }
        RefreshTokenService.Rotated rotated;
        try {
            rotated = refresh.rotate(raw);
        } catch (RefreshReuseException e) {
            log.warn("refresh token reuse detected; family revoked family_id={} user_id={}",
                    e.familyId(), e.userId());
            return unauthenticated(true);
        } catch (InvalidRefreshException e) {
            return unauthenticated(true);
        }
        // Roles come from users, so a role change applies at the next refresh.
        Optional<User> user = users.findById(rotated.userId());
        if (user.isEmpty()) { // account deleted since login
            refresh.revoke(rotated.raw());
            return unauthenticated(true);
        }
        return access(rotated.raw(), user.get().id(), user.get().role());
    }

    @PostMapping("/token/logout")
    ResponseEntity<Void> logout(@CookieValue(name = COOKIE_NAME, required = false) String raw) {
        refresh.revoke(raw);
        return ResponseEntity.noContent().header(HttpHeaders.SET_COOKIE, cookie("", Duration.ZERO).toString()).build();
    }

    private ResponseEntity<Map<String, Object>> access(String rawRefresh, java.util.UUID userId, String role) {
        JwtService.IssuedToken token = jwt.issue(new AuthenticatedPrincipal(userId, List.of(role)));
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("access_token", token.value());
        body.put("token_type", "Bearer");
        body.put("expires_in", JwtService.TTL.toSeconds());
        return ResponseEntity.ok()
                .header(HttpHeaders.SET_COOKIE, cookie(rawRefresh, RefreshTokenService.TTL).toString())
                .body(body);
    }

    /** Every refresh failure looks the same to the client; a bad token also clears the cookie. */
    private ResponseEntity<Map<String, Object>> unauthenticated(boolean clearCookie) {
        ResponseEntity.BodyBuilder b = ResponseEntity.status(HttpStatus.UNAUTHORIZED);
        if (clearCookie) {
            b.header(HttpHeaders.SET_COOKIE, cookie("", Duration.ZERO).toString());
        }
        return b.body(Map.of("error", "unauthenticated"));
    }

    private ResponseCookie cookie(String value, Duration maxAge) {
        return ResponseCookie.from(COOKIE_NAME, value)
                .path(COOKIE_PATH)
                .maxAge(maxAge)
                .httpOnly(true)
                .secure(secure)
                .sameSite("Strict")
                .build();
    }
}
