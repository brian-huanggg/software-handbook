package com.example.cookies;

import java.util.Map;

import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseCookie;
import org.springframework.http.ResponseEntity;
import org.springframework.web.HttpRequestMethodNotSupportedException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class LoginController {
    private final SessionStore store;

    public LoginController(SessionStore store) {
        this.store = store;
    }

    @PostMapping("/login")
    public ResponseEntity<Map<String, String>> login() {
        String id = store.create();

        ResponseCookie cookie = ResponseCookie.from("session_id", id)
                .path("/")
                .maxAge(SessionStore.TTL)
                .httpOnly(true)
                .secure(true)
                .sameSite("Lax")
                .build();

        return ResponseEntity.ok()
                .header(HttpHeaders.SET_COOKIE, cookie.toString())
                .body(Map.of("message", "Login successful, cookie has been set."));
    }

    // 非 POST 的 /login 一律回 404，與規格一致
    @ExceptionHandler(HttpRequestMethodNotSupportedException.class)
    public ResponseEntity<String> notFound() {
        return ResponseEntity.status(HttpStatus.NOT_FOUND).body("Not Found");
    }
}
