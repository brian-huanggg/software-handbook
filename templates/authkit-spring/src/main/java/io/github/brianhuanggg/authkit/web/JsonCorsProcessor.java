package io.github.brianhuanggg.authkit.web;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import org.springframework.http.server.ServerHttpResponse;
import org.springframework.http.server.ServletServerHttpResponse;
import org.springframework.web.cors.CorsConfiguration;
import org.springframework.web.cors.CorsUtils;
import org.springframework.web.cors.DefaultCorsProcessor;

/**
 * Spring's CORS processing, adjusted to the Go API's contract:
 * <ul>
 * <li>an approved preflight answers 204, not 200;</li>
 * <li>a rejected preflight answers a JSON 403;</li>
 * <li>a normal request from an unlisted origin is still served, without CORS
 * headers, so the browser will not expose it (unsafe methods are stopped by
 * the CSRF filter).</li>
 * </ul>
 */
public final class JsonCorsProcessor extends DefaultCorsProcessor {

    @Override
    public boolean processRequest(CorsConfiguration config, HttpServletRequest request,
            HttpServletResponse response) throws IOException {
        boolean preflight = CorsUtils.isPreFlightRequest(request);
        String origin = request.getHeader("Origin");
        if (!preflight && config != null && origin != null && config.checkOrigin(origin) == null) {
            // Keep the Vary headers the default processor would add.
            super.processRequest(null, request, response);
            return true;
        }
        boolean ok = super.processRequest(config, request, response);
        if (ok && preflight) {
            response.setStatus(HttpServletResponse.SC_NO_CONTENT);
        }
        return ok;
    }

    @Override
    protected void rejectRequest(ServerHttpResponse response) throws IOException {
        HttpServletResponse servlet = ((ServletServerHttpResponse) response).getServletResponse();
        ApiErrors.write(servlet, 403, "origin not allowed");
    }
}
