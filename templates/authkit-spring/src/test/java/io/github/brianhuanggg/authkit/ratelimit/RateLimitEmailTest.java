package io.github.brianhuanggg.authkit.ratelimit;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.clearInvocations;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import java.util.UUID;
import java.util.concurrent.atomic.AtomicInteger;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.bean.override.mockito.MockitoSpyBean;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.request.MockHttpServletRequestBuilder;

@AutoConfigureMockMvc
@TestPropertySource(properties = "authkit.rate-limit-email-burst=2")
class RateLimitEmailTest extends PostgresTestBase {

    static final String PASSWORD = "correct horse battery";
    static final AtomicInteger NEXT_IP = new AtomicInteger(1);

    @Autowired
    MockMvc mvc;

    @MockitoSpyBean
    PasswordEncoder encoder;

    /** A fresh client address per request, so only the email limit can trigger. */
    private static MockHttpServletRequestBuilder login(String path, String body) {
        String ip = "192.0.2." + (NEXT_IP.getAndIncrement() % 250 + 1);
        return post(path).with(r -> {
            r.setRemoteAddr(ip);
            return r;
        }).contentType(MediaType.APPLICATION_JSON).header("X-Requested-With", "XMLHttpRequest").content(body);
    }

    private static String creds(String email, String password) {
        return "{\"email\":\"%s\",\"password\":\"%s\"}".formatted(email, password);
    }

    @Test
    void loginIsLimitedPerEmailBeforeAnyHashing() throws Exception {
        String email = "u-" + UUID.randomUUID() + "@example.com"; // never registered
        mvc.perform(login("/session/login", creds(email, PASSWORD))).andExpect(status().isUnauthorized());
        mvc.perform(login("/session/login", creds(email, PASSWORD))).andExpect(status().isUnauthorized());
        clearInvocations(encoder);

        mvc.perform(login("/session/login", creds(email, PASSWORD)))
                .andExpect(status().isTooManyRequests())
                .andExpect(header().string("Retry-After", "30"))
                .andExpect(jsonPath("$.error").value("too many login attempts; try again later"));
        verify(encoder, never()).matches(any(), any());
        verify(encoder, never()).encode(any());
    }

    @Test
    void emailIsNormalizedAndSharedByBothLogins() throws Exception {
        String base = "case-" + UUID.randomUUID() + "@example.com";
        mvc.perform(login("/session/login", creds(base, PASSWORD))).andExpect(status().isUnauthorized());
        mvc.perform(login("/token/login", creds("  " + base.toUpperCase() + " ", PASSWORD)))
                .andExpect(status().isUnauthorized());
        mvc.perform(login("/token/login", creds(base, PASSWORD))).andExpect(status().isTooManyRequests());
        mvc.perform(login("/session/login", creds(base.toUpperCase(), PASSWORD))).andExpect(status().isTooManyRequests());
    }

    @Test
    void otherEmailsAreNotAffected() throws Exception {
        String a = "a-" + UUID.randomUUID() + "@example.com";
        for (int i = 0; i < 3; i++) {
            mvc.perform(login("/session/login", creds(a, PASSWORD)));
        }
        mvc.perform(login("/session/login", creds("b-" + UUID.randomUUID() + "@example.com", PASSWORD)))
                .andExpect(status().isUnauthorized());
    }

    /** A request the controller rejects with 400 must not spend the account's budget. */
    @Test
    void badBodiesDoNotSpendTheEmailLimit() throws Exception {
        String email = "bad-" + UUID.randomUUID() + "@example.com";
        for (int i = 0; i < 4; i++) {
            mvc.perform(login("/session/login", "{\"email\":\"%s\"}".formatted(email)))
                    .andExpect(status().isBadRequest());
            mvc.perform(login("/session/login", "{\"email\":\"%s\",\"password\":\"x\",\"extra\":1}".formatted(email)))
                    .andExpect(status().isBadRequest());
            mvc.perform(login("/session/login", "not json")).andExpect(status().isBadRequest());
        }
        mvc.perform(login("/session/login", creds(email, PASSWORD))).andExpect(status().isUnauthorized());
    }

    @Test
    void registerHasNoEmailLimit() throws Exception {
        String email = "reg-" + UUID.randomUUID() + "@example.com";
        for (int i = 0; i < 4; i++) {
            mvc.perform(login("/register", creds(email, PASSWORD))).andExpect(status().isAccepted());
        }
    }
}
