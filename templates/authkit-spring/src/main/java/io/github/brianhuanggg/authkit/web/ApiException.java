package io.github.brianhuanggg.authkit.web;

import org.springframework.http.HttpStatus;

/** An error with a status and a client-safe message, answered as {@code {"error": message}}. */
public class ApiException extends RuntimeException {

    private final HttpStatus status;

    public ApiException(HttpStatus status, String message) {
        super(message);
        this.status = status;
    }

    public HttpStatus status() {
        return status;
    }
}
