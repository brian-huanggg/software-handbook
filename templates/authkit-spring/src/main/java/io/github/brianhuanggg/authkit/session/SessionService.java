package io.github.brianhuanggg.authkit.session;

import io.github.brianhuanggg.authkit.auth.AuthenticatedPrincipal;
import io.github.brianhuanggg.authkit.config.AuthkitProperties;
import io.github.brianhuanggg.authkit.user.UserRepository;
import jakarta.servlet.http.Cookie;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.util.Base64;
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

    private static final int ID_BYTES = 32;
    private static final SecureRandom RANDOM = new SecureRandom();

    private final SessionStore store;
    private final UserRepository users;
    private final Clock clock;
    private final boolean secure;

    SessionService(SessionStore store, UserRepository users, Clock clock, AuthkitProperties props) {
        this.store = store;
        this.users = users;
        this.clock = clock;
        this.secure = !props.insecureCookies();
    }

    public static byte[] hashId(String rawId) {
        try {
            return MessageDigest.getInstance("SHA-256").digest(rawId.getBytes(StandardCharsets.UTF_8));
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException(e);
        }
    }

    /**
     * Creates a session for {@code userId} and sets the cookie. It always issues
     * a brand-new ID and revokes any session the request already carried, so an
     * ID planted before login (session fixation) never becomes authenticated.
     */
    public void login(HttpServletRequest request, HttpServletResponse response, UUID userId) {
        cookieValue(request).ifPresent(old -> store.delete(hashId(old)));
        byte[] random = new byte[ID_BYTES];
        RANDOM.nextBytes(random);
        String id = Base64.getUrlEncoder().withoutPadding().encodeToString(random);
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
        Optional<Session> found = store.find(hashId(id.get()));
        if (found.isEmpty()) {
            return Optional.empty();
        }
        Session session = found.get();
        if (!clock.instant().isBefore(session.expiresAt())) {
            store.delete(session.idHash());
            return Optional.empty();
        }
        // A deleted user means the session is no longer valid.
        return users.findById(session.userId())
                .map(u -> new AuthenticatedPrincipal(u.id(), List.of(u.role())));
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
