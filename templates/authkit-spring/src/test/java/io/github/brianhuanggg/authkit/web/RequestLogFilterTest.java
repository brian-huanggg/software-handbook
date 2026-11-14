package io.github.brianhuanggg.authkit.web;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.read.ListAppender;
import io.github.brianhuanggg.authkit.PostgresTestBase;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.stream.Collectors;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.slf4j.LoggerFactory;
import org.slf4j.MDC;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;

@AutoConfigureMockMvc
@TestPropertySource(properties = "authkit.rate-limit-ip-burst=2")
class RequestLogFilterTest extends PostgresTestBase {

    @Autowired
    MockMvc mvc;

    private final Logger root = (Logger) LoggerFactory.getLogger(Logger.ROOT_LOGGER_NAME);
    private ListAppender<ILoggingEvent> logs;

    @BeforeEach
    void capture() {
        logs = new ListAppender<>();
        logs.start();
        root.addAppender(logs);
        root.setLevel(Level.INFO);
    }

    @AfterEach
    void release() {
        root.detachAppender(logs);
    }

    private List<ILoggingEvent> access() {
        return logs.list.stream().filter(e -> e.getLoggerName().endsWith("RequestLogFilter")).toList();
    }

    private static Map<String, Object> fields(ILoggingEvent e) {
        return e.getKeyValuePairs().stream().collect(Collectors.toMap(kv -> kv.key, kv -> kv.value));
    }

    @Test
    void logsOneLinePerRequestWithItsFields() throws Exception {
        MvcResult r = mvc.perform(get("/token/me").queryParam("secret", "s3cr3t").with(req -> {
            req.setRemoteAddr("203.0.113.9");
            return req;
        })).andExpect(status().isUnauthorized()).andReturn();

        assertThat(access()).hasSize(1);
        ILoggingEvent line = access().get(0);
        Map<String, Object> f = fields(line);
        assertThat(f).containsEntry("method", "GET").containsEntry("path", "/token/me")
                .containsEntry("status", 401).containsEntry("ip", "203.0.113.9");
        assertThat((Long) f.get("bytes")).isEqualTo(r.getResponse().getContentAsByteArray().length).isPositive();
        assertThat((Long) f.get("duration_ms")).isGreaterThanOrEqualTo(0);
        assertThat(line.getFormattedMessage()).startsWith("request method=GET path=/token/me status=401");
        assertThat(line.getFormattedMessage()).doesNotContain("secret").doesNotContain("s3cr3t");
        assertThat(line.getMDCPropertyMap()).containsEntry("request_id", r.getResponse().getHeader("X-Request-Id"));
    }

    @Test
    void generatesAnIdWhenTheCallerSendsNone() throws Exception {
        String id = mvc.perform(get("/token/me")).andReturn().getResponse().getHeader("X-Request-Id");
        assertThat(id).matches("[0-9a-f]{16}");
    }

    @Test
    void reusesAValidCallerId() throws Exception {
        mvc.perform(get("/token/me").header("X-Request-Id", "abc-123.XYZ"))
                .andExpect(header().string("X-Request-Id", "abc-123.XYZ"));
        assertThat(access().get(0).getMDCPropertyMap()).containsEntry("request_id", "abc-123.XYZ");
    }

    @Test
    void replacesAnUnsafeCallerId() throws Exception {
        for (String bad : new String[] {"has space", "tab\there", "é-non-ascii", "x".repeat(65), "line\nbreak"}) {
            String id = mvc.perform(get("/token/me").header("X-Request-Id", bad)).andReturn()
                    .getResponse().getHeader("X-Request-Id");
            assertThat(id).as(bad).matches("[0-9a-f]{16}");
        }
        assertThat(RequestLogFilter.validId("x".repeat(64))).isTrue();
    }

    @Test
    void probesAreNotLogged() throws Exception {
        mvc.perform(get("/healthz")).andExpect(status().isOk());
        mvc.perform(get("/readyz")).andExpect(status().isOk());
        assertThat(access()).isEmpty();
    }

    /** The filter is outermost, so requests refused by a later filter are logged too. */
    @Test
    void logsRequestsRejectedByLaterFilters() throws Exception {
        // 403 from the CSRF filter (no X-Requested-With)
        mvc.perform(post("/register").contentType(MediaType.APPLICATION_JSON).content("{}"))
                .andExpect(status().isForbidden());
        // 413 from the body size limit
        mvc.perform(post("/register").header("X-Requested-With", "XMLHttpRequest")
                .contentType(MediaType.APPLICATION_JSON).content("{\"a\":\"" + "x".repeat(70_000) + "\"}"))
                .andExpect(status().isContentTooLarge());
        // 429 from the rate limiter (burst 2 per IP; two requests above did not reach it)
        String body = "{\"email\":\"u-" + UUID.randomUUID() + "@example.com\",\"password\":\"correct horse battery\"}";
        for (int i = 0; i < 3; i++) {
            mvc.perform(post("/register").header("X-Requested-With", "XMLHttpRequest").with(req -> {
                req.setRemoteAddr("198.51.100.77");
                return req;
            }).contentType(MediaType.APPLICATION_JSON).content(body));
        }

        assertThat(access().stream().map(e -> fields(e).get("status")).toList())
                .containsExactly(403, 413, 202, 202, 429);
    }

    @Test
    void controllerLogsCarryTheRequestId() throws Exception {
        String email = "dup-" + UUID.randomUUID() + "@example.com";
        String body = "{\"email\":\"%s\",\"password\":\"correct horse battery\"}".formatted(email);
        for (int i = 0; i < 2; i++) {
            mvc.perform(post("/register").header("X-Requested-With", "XMLHttpRequest").header("X-Request-Id", "req-" + i)
                    .contentType(MediaType.APPLICATION_JSON).content(body));
        }
        ILoggingEvent taken = logs.list.stream()
                .filter(e -> e.getFormattedMessage().contains("email already taken")).findFirst().orElseThrow();
        assertThat(taken.getMDCPropertyMap()).containsEntry("request_id", "req-1");
    }

    @Test
    void mdcIsClearedAfterTheRequest() throws Exception {
        mvc.perform(get("/token/me"));
        assertThat(MDC.get("request_id")).isNull();
    }
}
