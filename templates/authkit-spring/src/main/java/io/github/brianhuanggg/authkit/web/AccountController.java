package io.github.brianhuanggg.authkit.web;

import io.github.brianhuanggg.authkit.user.EmailTakenException;
import io.github.brianhuanggg.authkit.user.InvalidEmailException;
import io.github.brianhuanggg.authkit.user.UserRepository;
import java.nio.charset.StandardCharsets;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

@RestController
class AccountController {

    private static final Logger log = LoggerFactory.getLogger(AccountController.class);

    static final int MIN_PASSWORD_BYTES = 8;
    static final int MAX_PASSWORD_BYTES = 128; // bounds the work a single hash can be made to do

    record Credentials(String email, String password) {
    }

    private final UserRepository users;
    private final PasswordEncoder encoder;

    AccountController(UserRepository users, PasswordEncoder encoder) {
        this.users = users;
        this.encoder = encoder;
    }

    @PostMapping("/register")
    ResponseEntity<?> register(@RequestBody Credentials c) {
        if (c.email() == null || c.email().isEmpty()) {
            return error(HttpStatus.BAD_REQUEST, "email is required");
        }
        int n = c.password() == null ? 0 : c.password().getBytes(StandardCharsets.UTF_8).length;
        if (n < MIN_PASSWORD_BYTES || n > MAX_PASSWORD_BYTES) {
            return error(HttpStatus.BAD_REQUEST,
                    "password must be %d-%d bytes".formatted(MIN_PASSWORD_BYTES, MAX_PASSWORD_BYTES));
        }
        // Hash first, always: a new and an existing email cost the same work.
        String hash = encoder.encode(c.password());
        try {
            users.create(c.email(), hash);
        } catch (EmailTakenException e) {
            // Answer exactly like a new account, so /register cannot be used to
            // find out who has one. A real app emails the owner here
            // ("someone tried to sign up").
            log.info("register: email already taken");
        } catch (InvalidEmailException e) {
            return error(HttpStatus.BAD_REQUEST, "invalid email");
        }
        return registered();
    }

    /** The single response for both a new and an existing account. It carries no user ID. */
    private static ResponseEntity<?> registered() {
        return ResponseEntity.accepted().body(Map.of("status", "registration accepted"));
    }

    private static ResponseEntity<?> error(HttpStatus status, String message) {
        return ResponseEntity.status(status).body(Map.of("error", message));
    }
}
