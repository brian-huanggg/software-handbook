package io.github.brianhuanggg.authkit.web;

import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.IOException;
import org.springframework.security.core.AuthenticationException;
import org.springframework.security.web.AuthenticationEntryPoint;

/** 401 as {@code {"error":"unauthenticated"}}, with an optional {@code WWW-Authenticate} challenge. */
public final class JsonAuthenticationEntryPoint implements AuthenticationEntryPoint {

    private final String challenge;

    /** @param challenge value for {@code WWW-Authenticate}; null sends none */
    public JsonAuthenticationEntryPoint(String challenge) {
        this.challenge = challenge;
    }

    @Override
    public void commence(HttpServletRequest request, HttpServletResponse response,
            AuthenticationException authException) throws IOException {
        if (challenge != null) {
            response.setHeader("WWW-Authenticate", challenge);
        }
        ApiErrors.write(response, 401, "unauthenticated");
    }
}
