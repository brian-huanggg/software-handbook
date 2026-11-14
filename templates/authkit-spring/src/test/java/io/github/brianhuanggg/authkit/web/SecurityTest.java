package io.github.brianhuanggg.authkit.web;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.clearInvocations;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.user;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.options;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Import;
import org.springframework.core.annotation.Order;
import org.springframework.http.MediaType;
import org.springframework.security.config.annotation.web.builders.HttpSecurity;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.security.web.SecurityFilterChain;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.context.bean.override.mockito.MockitoSpyBean;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockHttpServletRequestBuilder;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@AutoConfigureMockMvc
@TestPropertySource(properties = "authkit.cors-origins=http://localhost:3000")
@Import(SecurityTest.TestEndpoints.class)
class SecurityTest extends PostgresTestBase {

    static final String ORIGIN = "http://localhost:3000";
    static final String EVIL = "http://evil.example";

    @RestController
    static class Boom {
        @GetMapping("/test/boom")
        String boom() {
            throw new IllegalStateException("secret internal detail");
        }
    }

    @TestConfiguration(proxyBeanMethods = false)
    @Import(Boom.class)
    static class TestEndpoints {
        /** A chain with a role rule, wired with the production 401/403 handlers. */
        @Bean
        @Order(0)
        SecurityFilterChain adminOnly(HttpSecurity http) throws Exception {
            http.securityMatcher("/test/admin")
                    .authorizeHttpRequests(a -> a.anyRequest().hasRole("admin"))
                    .exceptionHandling(e -> e
                            .authenticationEntryPoint(new JsonAuthenticationEntryPoint(null))
                            .accessDeniedHandler(new JsonAccessDeniedHandler()));
            return http.build();
        }
    }

    @Autowired
    MockMvc mvc;

    @MockitoSpyBean
    PasswordEncoder encoder;

    private static MockHttpServletRequestBuilder register(String json) {
        return register(json, ORIGIN);
    }

    private static MockHttpServletRequestBuilder register(String json, String origin) {
        return post("/register").contentType(MediaType.APPLICATION_JSON).content(json)
                .header("X-Requested-With", "XMLHttpRequest").header("Origin", origin);
    }

    private static String creds(String email, String password) {
        return "{\"email\":\"%s\",\"password\":\"%s\"}".formatted(email, password);
    }

    private static String newEmail() {
        return "u-" + UUID.randomUUID() + "@example.com";
    }

    // ---- error bodies ----

    @Test
    void unauthenticatedIs401Json() throws Exception {
        mvc.perform(get("/session/me"))
                .andExpect(status().isUnauthorized())
                .andExpect(header().string("Cache-Control", "no-store"))
                .andExpect(header().doesNotExist("WWW-Authenticate"))
                .andExpect(jsonPath("$.error").value("unauthenticated"));
    }

    @Test
    void tokenModeChallengesWithBearer() throws Exception {
        mvc.perform(get("/token/me"))
                .andExpect(status().isUnauthorized())
                .andExpect(header().string("WWW-Authenticate", "Bearer realm=\"authkit\""))
                .andExpect(jsonPath("$.error").value("unauthenticated"));
    }

    @Test
    void wrongRoleIs403Json() throws Exception {
        mvc.perform(get("/test/admin").with(user("bob").roles("user")))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error").value("forbidden"));
    }

    @Test
    void unhandledExceptionIsGeneric500() throws Exception {
        mvc.perform(get("/test/boom"))
                .andExpect(status().isInternalServerError())
                .andExpect(header().string("Cache-Control", "no-store"))
                .andExpect(jsonPath("$.error").value("internal server error"))
                .andExpect(content().string(org.hamcrest.Matchers.not(
                        org.hamcrest.Matchers.containsString("secret"))));
    }

    @Test
    void unknownPathIsJson404() throws Exception {
        mvc.perform(get("/nope"))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.error").value("not found"));
    }

    @Test
    void wrongMethodIsJson405() throws Exception {
        mvc.perform(get("/register"))
                .andExpect(status().isMethodNotAllowed())
                .andExpect(jsonPath("$.error").value("method not allowed"));
    }

    // ---- CORS ----

    @Test
    void preflightFromAllowedOrigin() throws Exception {
        mvc.perform(options("/token/refresh").header("Origin", ORIGIN)
                        .header("Access-Control-Request-Method", "POST")
                        .header("Access-Control-Request-Headers", "x-requested-with,content-type"))
                .andExpect(status().isNoContent())
                .andExpect(header().string("Access-Control-Allow-Origin", ORIGIN))
                .andExpect(header().string("Access-Control-Allow-Credentials", "true"))
                .andExpect(header().string("Access-Control-Max-Age", "600"))
                .andExpect(header().string("Access-Control-Allow-Headers",
                        org.hamcrest.Matchers.containsStringIgnoringCase("x-requested-with")));
    }

    @Test
    void preflightFromOtherOriginIs403() throws Exception {
        mvc.perform(options("/token/refresh").header("Origin", EVIL)
                        .header("Access-Control-Request-Method", "POST"))
                .andExpect(status().isForbidden())
                .andExpect(header().doesNotExist("Access-Control-Allow-Origin"))
                .andExpect(jsonPath("$.error").value("origin not allowed"));
    }

    @Test
    void realResponseCarriesCorsHeadersAndNoWildcard() throws Exception {
        mvc.perform(register(creds(newEmail(), "password-123")))
                .andExpect(status().isAccepted())
                .andExpect(header().string("Access-Control-Allow-Origin", ORIGIN))
                .andExpect(header().string("Access-Control-Allow-Credentials", "true"));
    }

    @Test
    void safeRequestFromOtherOriginIsServedWithoutCorsHeaders() throws Exception {
        mvc.perform(get("/healthz").header("Origin", EVIL))
                .andExpect(status().isOk())
                .andExpect(header().doesNotExist("Access-Control-Allow-Origin"));
    }

    // ---- CSRF ----

    @Test
    void unsafeRequestWithoutHeaderIs403() throws Exception {
        for (String path : new String[] {"/register", "/session/login", "/token/login", "/token/refresh",
                "/session/logout", "/token/logout"}) {
            MvcResult r = mvc.perform(post(path).contentType(MediaType.APPLICATION_JSON)
                            .content(creds(newEmail(), "password-123")).header("Origin", ORIGIN))
                    .andExpect(status().isForbidden())
                    .andExpect(jsonPath("$.error").value("missing X-Requested-With header"))
                    .andReturn();
            assertThat(r.getResponse().getCookies()).as(path).isEmpty();
        }
    }

    @Test
    void unsafeRequestFromOtherOriginIs403() throws Exception {
        mvc.perform(register(creds(newEmail(), "password-123"), EVIL))
                .andExpect(status().isForbidden())
                .andExpect(jsonPath("$.error").value("forbidden origin"));
    }

    @Test
    void unsafeRequestWithoutOriginButWithHeaderPasses() throws Exception {
        mvc.perform(post("/register").contentType(MediaType.APPLICATION_JSON)
                        .content(creds(newEmail(), "password-123")).header("X-Requested-With", "XMLHttpRequest"))
                .andExpect(status().isAccepted());
    }

    // ---- /register ----

    @Test
    void newAndExistingEmailGetIdenticalResponsesAndEqualHashing() throws Exception {
        String email = newEmail();
        clearInvocations(encoder);

        MvcResult fresh = mvc.perform(register(creds(email, "password-123")))
                .andExpect(status().isAccepted()).andReturn();
        MvcResult again = mvc.perform(register(creds(email, "password-456")))
                .andExpect(status().isAccepted()).andReturn();
        MvcResult upper = mvc.perform(register(creds(email.toUpperCase(), "password-789")))
                .andExpect(status().isAccepted()).andReturn();

        assertThat(again.getResponse().getContentAsString()).isEqualTo(fresh.getResponse().getContentAsString());
        assertThat(upper.getResponse().getContentAsString()).isEqualTo(fresh.getResponse().getContentAsString());
        assertThat(fresh.getResponse().getContentAsString()).contains("registration accepted");
        // One real Argon2 hash per request, new account or not.
        verify(encoder, times(3)).encode(org.mockito.ArgumentMatchers.<CharSequence>any());
        // No session cookie: nothing created an HttpSession.
        assertThat(fresh.getResponse().getCookies()).isEmpty();
    }

    @Test
    void registerValidation() throws Exception {
        mvc.perform(register("{\"password\":\"password-123\"}"))
                .andExpect(status().isBadRequest()).andExpect(jsonPath("$.error").value("email is required"));
        mvc.perform(register(creds(newEmail(), "short")))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("password must be 8-128 bytes"));
        mvc.perform(register(creds(newEmail(), "x".repeat(129))))
                .andExpect(status().isBadRequest());
        mvc.perform(register(creds("not-an-email", "password-123")))
                .andExpect(status().isBadRequest()).andExpect(jsonPath("$.error").value("invalid email"));
    }

    @Test
    void registerRejectsBadJson() throws Exception {
        for (String body : new String[] {
                "{\"email\":\"a@example.com\",\"password\":\"password-123\",\"extra\":1}", // unknown field
                "{\"email\":\"a@example.com\",\"password\":\"password-123\"}{}", // trailing value
                "{\"email\":", // truncated
                "" }) {
            mvc.perform(register(body))
                    .andExpect(status().isBadRequest())
                    .andExpect(jsonPath("$.error").value("invalid JSON body"));
        }
    }

    @Test
    void registerNeverCalledEncoderOnBadInput() throws Exception {
        clearInvocations(encoder);
        mvc.perform(register(creds(newEmail(), "short"))).andExpect(status().isBadRequest());
        verify(encoder, times(0)).matches(org.mockito.ArgumentMatchers.<CharSequence>any(), anyString());
    }
}
