package io.github.brianhuanggg.authkit.web;

import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.util.Set;
import org.springframework.web.filter.OncePerRequestFilter;

/**
 * Protects cookie-authenticated endpoints against cross-site request forgery
 * with two independent checks on unsafe methods (anything but GET, HEAD, OPTIONS):
 * <ol>
 * <li>A custom header is required. A cross-site page cannot add one without a
 * CORS preflight, and CORS only approves the listed origins. HTML forms cannot
 * set it at all.</li>
 * <li>If the browser sent an {@code Origin}, it must be an allowed origin. There
 * is no "same host as the API" fallback: behind a proxy the host is often the
 * internal one and it ignores the scheme. A page served from the API's own
 * origin must be listed like any other.</li>
 * </ol>
 * Safe methods must never change state, since they are not checked. Not a
 * bean: the security chain adds it, so Boot does not also register it as a
 * servlet filter.
 */
public final class CsrfHeaderFilter extends OncePerRequestFilter {

    public static final String HEADER = "X-Requested-With";
    public static final String VALUE = "XMLHttpRequest";

    private static final Set<String> SAFE_METHODS = Set.of("GET", "HEAD", "OPTIONS");

    private final Set<String> allowedOrigins;

    public CsrfHeaderFilter(java.util.Collection<String> allowedOrigins) {
        this.allowedOrigins = Set.copyOf(allowedOrigins);
    }

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        if (SAFE_METHODS.contains(request.getMethod())) {
            chain.doFilter(request, response);
            return;
        }
        String origin = request.getHeader("Origin");
        if (origin != null && !origin.isEmpty() && !allowedOrigins.contains(origin)) {
            ApiErrors.write(response, 403, "forbidden origin");
            return;
        }
        if (!VALUE.equals(request.getHeader(HEADER))) {
            ApiErrors.write(response, 403, "missing " + HEADER + " header");
            return;
        }
        chain.doFilter(request, response);
    }
}
