package io.github.brianhuanggg.authkit.ratelimit;

import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.clearInvocations;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import java.util.UUID;
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
@TestPropertySource(properties = "authkit.rate-limit-ip-burst=3")
class RateLimitIpTest extends PostgresTestBase {

    static final String PASSWORD = "correct horse battery";

    @Autowired
    MockMvc mvc;

    @MockitoSpyBean
    PasswordEncoder encoder;

    /** Each test uses its own client address, so they never share a bucket. */
    private static MockHttpServletRequestBuilder from(MockHttpServletRequestBuilder b, String ip) {
        b.with(r -> {
            r.setRemoteAddr(ip);
            return r;
        });
        return b;
    }

    private static MockHttpServletRequestBuilder json(String path, String ip, String email, String password) {
        return from(post(path), ip).contentType(MediaType.APPLICATION_JSON)
                .header("X-Requested-With", "XMLHttpRequest")
                .content("{\"email\":\"%s\",\"password\":\"%s\"}".formatted(email, password));
    }

    private static String email() {
        return "u-" + UUID.randomUUID() + "@example.com";
    }

    @Test
    void registerIsLimitedPerIpBeforeAnyHashing() throws Exception {
        clearInvocations(encoder);
        for (int i = 0; i < 3; i++) {
            mvc.perform(json("/register", "198.51.100.1", email(), PASSWORD)).andExpect(status().isAccepted());
        }
        verify(encoder, times(3)).encode(anyString());

        mvc.perform(json("/register", "198.51.100.1", email(), PASSWORD))
                .andExpect(status().isTooManyRequests())
                .andExpect(header().string("Retry-After", "3"))
                .andExpect(header().string("Cache-Control", "no-store"))
                .andExpect(jsonPath("$.error").value("too many requests"));
        verify(encoder, times(3)).encode(anyString()); // the 4th request hashed nothing
    }

    @Test
    void bothLoginsAndRegisterShareTheIpBucket() throws Exception {
        String ip = "198.51.100.2";
        mvc.perform(json("/register", ip, email(), PASSWORD)).andExpect(status().isAccepted());
        mvc.perform(json("/session/login", ip, email(), PASSWORD)).andExpect(status().isUnauthorized());
        mvc.perform(json("/token/login", ip, email(), PASSWORD)).andExpect(status().isUnauthorized());
        clearInvocations(encoder);

        mvc.perform(json("/session/login", ip, email(), PASSWORD)).andExpect(status().isTooManyRequests());
        mvc.perform(json("/token/login", ip, email(), PASSWORD)).andExpect(status().isTooManyRequests());
        mvc.perform(json("/register", ip, email(), PASSWORD)).andExpect(status().isTooManyRequests());
        verifyNoInteractions(encoder); // no encode, no matches: refused before any hashing
    }

    @Test
    void otherAddressesAreNotAffected() throws Exception {
        for (int i = 0; i < 4; i++) {
            mvc.perform(json("/register", "198.51.100.3", email(), PASSWORD));
        }
        mvc.perform(json("/register", "198.51.100.3", email(), PASSWORD)).andExpect(status().isTooManyRequests());
        mvc.perform(json("/register", "198.51.100.4", email(), PASSWORD)).andExpect(status().isAccepted());
    }

    @Test
    void ipv6AddressesInOneSlash64ShareABucket() throws Exception {
        for (int i = 1; i <= 3; i++) {
            mvc.perform(json("/register", "2001:db8:aa:bb::" + i, email(), PASSWORD)).andExpect(status().isAccepted());
        }
        mvc.perform(json("/register", "2001:db8:aa:bb:ffff::9", email(), PASSWORD))
                .andExpect(status().isTooManyRequests());
        mvc.perform(json("/register", "2001:db8:aa:cc::1", email(), PASSWORD)).andExpect(status().isAccepted());
    }

    @Test
    void otherEndpointsAreNotLimited() throws Exception {
        String ip = "198.51.100.5";
        for (int i = 0; i < 6; i++) {
            mvc.perform(from(get("/healthz"), ip)).andExpect(status().isOk());
            mvc.perform(from(post("/session/logout"), ip).header("X-Requested-With", "XMLHttpRequest"))
                    .andExpect(status().isNoContent());
            mvc.perform(from(get("/token/me"), ip)).andExpect(status().isUnauthorized());
        }
    }
}
