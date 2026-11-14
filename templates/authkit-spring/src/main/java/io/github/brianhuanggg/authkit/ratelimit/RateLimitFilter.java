package io.github.brianhuanggg.authkit.ratelimit;

import io.github.brianhuanggg.authkit.web.ApiErrors;
import io.github.brianhuanggg.authkit.web.Credentials;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.util.Locale;
import org.springframework.web.filter.OncePerRequestFilter;
import tools.jackson.core.JacksonException;
import tools.jackson.databind.DeserializationFeature;
import tools.jackson.databind.json.JsonMapper;

/**
 * Answers 429 with {@code Retry-After} before any password hash runs. The
 * client IP is limited on {@code /register} and both logins; the account
 * (lower-cased email) is limited on both logins, whether or not it exists, so
 * the limit itself reveals nothing. The trade-off: an attacker can slow down
 * one victim's logins.
 *
 * <p>The email comes from the body, which {@code BodySizeLimitFilter} has
 * already capped and buffered, so the controller can still read it. Not a
 * bean: each security chain adds its own instance, all sharing one
 * {@link RateLimits}.
 */
public final class RateLimitFilter extends OncePerRequestFilter {

    // Same rules as the controllers: unknown fields and trailing tokens are errors.
    // A body they would reject with 400 does not spend the email limit.
    private static final JsonMapper JSON = JsonMapper.builder()
            .enable(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES)
            .enable(DeserializationFeature.FAIL_ON_TRAILING_TOKENS)
            .build();

    private final RateLimits limits;

    public RateLimitFilter(RateLimits limits) {
        this.limits = limits;
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        boolean login = false;
        if ("POST".equals(request.getMethod())) {
            // The firewall in front of this filter has already rejected encoded or ambiguous paths.
            String path = request.getRequestURI().substring(request.getContextPath().length());
            login = path.equals("/session/login") || path.equals("/token/login");
            if (login || path.equals("/register")) {
                Limiter.Decision d = limits.perIp().allow(ClientIp.of(request));
                if (!d.allowed()) {
                    reject(response, d, "too many requests");
                    return;
                }
            }
        }
        if (login) {
            String email = loginEmail(request);
            if (email != null) {
                Limiter.Decision d = limits.perEmail().allow(email);
                if (!d.allowed()) {
                    reject(response, d, "too many login attempts; try again later");
                    return;
                }
            }
        }
        chain.doFilter(request, response);
    }

    /** The normalized email, or null when the controller will answer 400 anyway. */
    private static String loginEmail(HttpServletRequest request) throws IOException {
        Credentials c;
        try {
            c = JSON.readValue(request.getInputStream().readAllBytes(), Credentials.class);
        } catch (JacksonException e) {
            return null;
        }
        if (c == null || c.email() == null || c.email().isEmpty() || c.password() == null || c.password().isEmpty()) {
            return null;
        }
        return c.email().strip().toLowerCase(Locale.ROOT);
    }

    private static void reject(HttpServletResponse response, Limiter.Decision d, String message) throws IOException {
        long seconds = (d.retryAfter().toNanos() + 999_999_999L) / 1_000_000_000L; // round up
        response.setHeader("Retry-After", Long.toString(Math.max(1, seconds)));
        ApiErrors.write(response, 429, message);
    }
}
