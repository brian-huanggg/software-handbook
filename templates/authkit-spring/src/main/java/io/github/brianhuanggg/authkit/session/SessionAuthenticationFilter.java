package io.github.brianhuanggg.authkit.session;

import io.github.brianhuanggg.authkit.web.ApiErrors;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.security.core.context.SecurityContext;
import org.springframework.security.core.context.SecurityContextHolder;
import org.springframework.web.filter.OncePerRequestFilter;

/**
 * Reads the {@code sid} cookie and, if it is a live session, puts an
 * {@code AuthenticatedPrincipal} in the security context. A missing or invalid
 * cookie just leaves the request anonymous: the entry point answers 401 where
 * authentication is required. Not a bean: the session chain adds it.
 */
public final class SessionAuthenticationFilter extends OncePerRequestFilter {

    private static final Logger log = LoggerFactory.getLogger(SessionAuthenticationFilter.class);

    private final SessionService sessions;

    public SessionAuthenticationFilter(SessionService sessions) {
        this.sessions = sessions;
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        try {
            sessions.authenticate(request).ifPresent(principal -> {
                SecurityContext context = SecurityContextHolder.createEmptyContext();
                context.setAuthentication(principal.toAuthentication());
                SecurityContextHolder.setContext(context);
            });
        } catch (RuntimeException e) {
            log.error("authenticate session", e);
            ApiErrors.write(response, 500, "internal server error");
            return;
        }
        chain.doFilter(request, response);
    }
}
