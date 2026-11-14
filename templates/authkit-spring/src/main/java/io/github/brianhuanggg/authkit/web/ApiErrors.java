package io.github.brianhuanggg.authkit.web;

import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import java.util.Map;
import tools.jackson.databind.json.JsonMapper;

/**
 * The one error shape a client sees, from filters, handlers and controllers
 * alike: {@code {"error": "..."}}. The message reaches the client, so it must
 * never contain internal details.
 */
public final class ApiErrors {

    private static final JsonMapper JSON = JsonMapper.builder().build();

    private ApiErrors() {
    }

    public static void write(HttpServletResponse response, int status, String message) throws IOException {
        response.setStatus(status);
        response.setContentType("application/json");
        // Responses carry tokens, cookies or per-user data: never cached.
        response.setHeader("Cache-Control", "no-store");
        response.setHeader("X-Content-Type-Options", "nosniff");
        response.getOutputStream().write(JSON.writeValueAsBytes(Map.of("error", message)));
    }
}
