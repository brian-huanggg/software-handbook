package io.github.brianhuanggg.authkit.web;

import io.github.brianhuanggg.authkit.auth.AccountUserDetails;
import io.github.brianhuanggg.authkit.auth.LoginAuthenticator;
import io.github.brianhuanggg.authkit.session.SessionService;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.util.Map;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RestController;

/** Session mode: server-side state, cookie credential. */
@RestController
class SessionController {

    private final LoginAuthenticator logins;
    private final SessionService sessions;

    SessionController(LoginAuthenticator logins, SessionService sessions) {
        this.logins = logins;
        this.sessions = sessions;
    }

    @PostMapping("/session/login")
    Map<String, String> login(@RequestBody Credentials c, HttpServletRequest request, HttpServletResponse response) {
        AccountUserDetails user = logins.authenticate(c);
        sessions.login(request, response, user.id());
        return Map.of("user_id", user.id().toString());
    }

    @PostMapping("/session/logout")
    ResponseEntity<Void> logout(HttpServletRequest request, HttpServletResponse response) {
        sessions.logout(request, response);
        return ResponseEntity.noContent().build();
    }
}
