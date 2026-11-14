package io.github.brianhuanggg.authkit.session;

import io.github.brianhuanggg.authkit.auth.AuthenticatedPrincipal;
import io.github.brianhuanggg.authkit.config.AuthkitProperties;
import io.github.brianhuanggg.authkit.crypto.OpaqueIds;
import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.Optional;
import java.util.UUID;
import org.springframework.http.HttpHeaders;
import org.springframework.http.ResponseCookie;
import org.springframework.stereotype.Service;

/**
 * Issues, validates and revokes session cookies. The raw ID exists only in the
 * client's cookie; the store sees its SHA-256, so a leaked database cannot be
 * replayed as valid sessions.
 */
@Service
public class SessionService {

    public static final String COOKIE_NAME = "sid";
    public static final Duration TTL = Duration.ofHours(24);

    private final SessionStore store;
    private final Clock clock;
    private final boolean secure;

    SessionService(SessionStore store, Clock clock, AuthkitProperties props) {
        this.store = store;
        this.clock = clock;
        this.secure = !props.insecureCookies();
    }

    private static byte[] hashId(String rawId) {
        return OpaqueIds.hash(rawId);
    }

    /**
     * Creates a session for {@code userId} and sets the cookie. It always issues
     * a brand-new ID and revokes any session the request already carried, so an
     * ID planted before login (session fixation) never becomes authenticated.
     */
    public void login(HttpServletRequest request, HttpServletResponse response, UUID userId) {
        cookieValue(request).ifPresent(old -> store.delete(hashId(old)));
        String id = OpaqueIds.generate();
        Instant expires = clock.instant().plus(TTL);
        store.create(new Session(hashId(id), userId, expires));
        response.addHeader(HttpHeaders.SET_COOKIE, cookie(id, TTL).toString());
    }

    /** Revokes the request's session (if any) and clears the cookie. */
    public void logout(HttpServletRequest request, HttpServletResponse response) {
        response.addHeader(HttpHeaders.SET_COOKIE, cookie("", Duration.ZERO).toString());
        cookieValue(request).ifPresent(id -> store.delete(hashId(id)));
    }

    /**
     * Resolves the session cookie to a principal, loading roles from
     * {@code users} on every call so a role change applies on the next request.
     * Every kind of invalid cookie yields empty; infrastructure errors propagate.
     */
    public Optional<AuthenticatedPrincipal> authenticate(HttpServletRequest request) {
        Optional<String> id = cookieValue(request);
        if (id.isEmpty()) {
            return Optional.empty();
        }
        // Session and role in one lookup; a deleted user means no session.
        Optional<SessionUser> found = store.findWithRole(hashId(id.get()));
        if (found.isEmpty()) {
            return Optional.empty();
        }
        Session session = found.get().session();
        if (!clock.instant().isBefore(session.expiresAt())) {
            store.delete(session.idHash());
            return Optional.empty();
        }
        return Optional.of(new AuthenticatedPrincipal(session.userId(), List.of(found.get().role())));
    }

    private static Optional<String> cookieValue(HttpServletRequest request) {
        Cookie[] cookies = request.getCookies();
        if (cookies == null) {
            return Optional.empty();
        }
        for (Cookie c : cookies) {
            if (c.getName().equals(COOKIE_NAME)) {
                return c.getValue() == null || c.getValue().isEmpty() ? Optional.empty() : Optional.of(c.getValue());
            }
        }
        return Optional.empty();
    }

    private ResponseCookie cookie(String value, Duration maxAge) {
        return ResponseCookie.from(COOKIE_NAME, value)
                .path("/")
                .maxAge(maxAge)
                .httpOnly(true)
                .secure(secure)
                .sameSite("Lax")
                .build();
    }
}
