package io.github.brianhuanggg.authkit.web;

import org.springframework.http.MediaType;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
class HealthController {

    /** Liveness: never touches the database. */
    @GetMapping(path = "/healthz", produces = MediaType.TEXT_PLAIN_VALUE)
    String healthz() {
        return "ok";
    }
}
