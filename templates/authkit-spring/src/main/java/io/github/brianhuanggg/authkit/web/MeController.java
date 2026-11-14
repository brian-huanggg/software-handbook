package io.github.brianhuanggg.authkit.web;

import io.github.brianhuanggg.authkit.auth.AuthenticatedPrincipal;
import java.util.Map;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

/** The protected endpoint of both modes; the chain decides which credential is accepted. */
@RestController
class MeController {

    @GetMapping({"/session/me", "/token/me"})
    Map<String, Object> me(@AuthenticationPrincipal AuthenticatedPrincipal principal) {
        return Map.of("user_id", principal.userId().toString(), "roles", principal.roles());
    }
}
