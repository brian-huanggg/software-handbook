package io.github.brianhuanggg.authkit.web;

import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.HttpStatusCode;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.http.converter.HttpMessageNotReadableException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.context.request.WebRequest;
import org.springframework.web.servlet.mvc.method.annotation.ResponseEntityExceptionHandler;

/**
 * Every MVC error becomes {@code {"error": "..."}}. Spring's own exceptions
 * (400, 404, 405, 415, ...) keep their status; anything else is a generic 500
 * that never leaks the exception message.
 */
@RestControllerAdvice
class ApiExceptionHandler extends ResponseEntityExceptionHandler {

    private static final Logger log = LoggerFactory.getLogger(ApiExceptionHandler.class);

    @ExceptionHandler(ApiException.class)
    ResponseEntity<Object> api(ApiException ex) {
        return json(ex.status(), ex.getMessage());
    }

    @ExceptionHandler(Exception.class)
    ResponseEntity<Object> internal(Exception ex) {
        log.error("unhandled error", ex);
        return json(HttpStatus.INTERNAL_SERVER_ERROR, "internal server error");
    }

    @Override
    protected ResponseEntity<Object> handleExceptionInternal(Exception ex, Object body, HttpHeaders headers,
            HttpStatusCode status, WebRequest request) {
        if (status.is5xxServerError()) {
            log.error("server error", ex);
            return json(status, "internal server error");
        }
        String message = ex instanceof HttpMessageNotReadableException
                ? "invalid JSON body"
                : message(status);
        return json(status, message);
    }

    private static String message(HttpStatusCode status) {
        HttpStatus known = HttpStatus.resolve(status.value());
        return known == null ? "error" : known.getReasonPhrase().toLowerCase();
    }

    private static ResponseEntity<Object> json(HttpStatusCode status, String message) {
        return ResponseEntity.status(status).contentType(MediaType.APPLICATION_JSON)
                .body(Map.of("error", message));
    }
}
