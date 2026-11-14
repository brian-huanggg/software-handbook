package io.github.brianhuanggg.authkit.web;

import java.sql.Connection;
import javax.sql.DataSource;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
class ReadinessController {

    private static final Logger log = LoggerFactory.getLogger(ReadinessController.class);
    private static final int PING_TIMEOUT_SECONDS = 2;

    private final DataSource dataSource;

    ReadinessController(DataSource dataSource) {
        this.dataSource = dataSource;
    }

    /** Readiness: {@code 200} when the database answers a ping, else {@code 503}. */
    @GetMapping(path = "/readyz", produces = MediaType.ALL_VALUE)
    ResponseEntity<?> readyz() {
        try (Connection c = dataSource.getConnection()) {
            if (c.isValid(PING_TIMEOUT_SECONDS)) {
                return ResponseEntity.ok().contentType(MediaType.TEXT_PLAIN).body("ok");
            }
            log.warn("not ready: database ping failed");
        } catch (Exception e) {
            log.warn("not ready", e);
        }
        return ResponseEntity.status(503).contentType(MediaType.APPLICATION_JSON)
                .body(java.util.Map.of("error", "not ready"));
    }
}
