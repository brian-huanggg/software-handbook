package io.github.brianhuanggg.authkit.session;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyString;
import static org.mockito.Mockito.clearInvocations;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import io.github.brianhuanggg.authkit.user.UserRepository;
import jakarta.servlet.http.Cookie;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.test.context.bean.override.mockito.MockitoSpyBean;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockHttpServletRequestBuilder;

@AutoConfigureMockMvc
class SessionFlowTest extends PostgresTestBase {

    static final String PASSWORD = "correct horse battery";

    @Autowired
    MockMvc mvc;
    @Autowired
    UserRepository users;
    @Autowired
    SessionStore store;
    @Autowired
    JdbcClient jdbc;
    @MockitoSpyBean
    PasswordEncoder encoder;

    private static MockHttpServletRequestBuilder json(MockHttpServletRequestBuilder b, String body) {
        return b.contentType(MediaType.APPLICATION_JSON).content(body).header("X-Requested-With", "XMLHttpRequest");
    }

    private static String creds(String email, String password) {
        return "{\"email\":\"%s\",\"password\":\"%s\"}".formatted(email, password);
    }

    /** Registers through the API, as a real client would. */
    private String newUser() throws Exception {
        String email = "u-" + UUID.randomUUID() + "@example.com";
        mvc.perform(json(post("/register"), creds(email, PASSWORD))).andExpect(status().isAccepted());
        return email;
    }

    private MvcResult login(String email, Cookie... cookies) throws Exception {
        MockHttpServletRequestBuilder request = json(post("/session/login"), creds(email, PASSWORD));
        if (cookies.length > 0) {
            request.cookie(cookies);
        }
        return mvc.perform(request).andExpect(status().isOk()).andReturn();
    }

    private static String sid(MvcResult login) {
        String header = login.getResponse().getHeaders(HttpHeaders.SET_COOKIE).stream()
                .filter(h -> h.startsWith("sid=")).findFirst().orElseThrow();
        return header.substring(4, header.indexOf(';'));
    }

    @Test
    void loginSetsHardenedCookieAndMeWorks() throws Exception {
        String email = newUser();
        MvcResult res = login(email);

        String setCookie = res.getResponse().getHeader(HttpHeaders.SET_COOKIE);
        assertThat(setCookie).startsWith("sid=").contains("HttpOnly", "Secure", "SameSite=Lax", "Path=/",
                "Max-Age=86400");
        // Our cookie only: nothing created an HttpSession.
        assertThat(res.getResponse().getHeaders(HttpHeaders.SET_COOKIE)).hasSize(1);
        UUID id = users.findByEmail(email).orElseThrow().id();
        assertThat(res.getResponse().getContentAsString()).contains(id.toString());

        mvc.perform(get("/session/me").cookie(new Cookie("sid", sid(res))))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.user_id").value(id.toString()))
                .andExpect(jsonPath("$.roles[0]").value("user"));
    }

    @Test
    void storesOnlyTheHashOfTheId() throws Exception {
        String raw = sid(login(newUser()));
        assertThat(store.find(SessionService.hashId(raw))).isPresent();
        assertThat(jdbc.sql("SELECT count(*) FROM sessions WHERE id_hash = convert_to(:raw, 'UTF8')")
                .param("raw", raw).query(Integer.class).single()).isZero();
    }

    @Test
    void unknownAndWrongPasswordLookTheSame() throws Exception {
        String email = newUser();
        MvcResult wrong = mvc.perform(json(post("/session/login"), creds(email, "not the password")))
                .andExpect(status().isUnauthorized()).andReturn();
        MvcResult unknown = mvc.perform(json(post("/session/login"),
                        creds("ghost-" + UUID.randomUUID() + "@example.com", PASSWORD)))
                .andExpect(status().isUnauthorized()).andReturn();
        assertThat(unknown.getResponse().getContentAsString()).isEqualTo(wrong.getResponse().getContentAsString())
                .contains("invalid email or password");
        assertThat(wrong.getResponse().getHeaders(HttpHeaders.SET_COOKIE)).isEmpty();
    }

    @Test
    void missingFieldsAre400() throws Exception {
        mvc.perform(json(post("/session/login"), "{\"email\":\"a@example.com\"}"))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("email and password are required"));
    }

    @Test
    void overlongPasswordIs401WithoutHashing() throws Exception {
        String email = newUser();
        clearInvocations(encoder);
        mvc.perform(json(post("/session/login"), creds(email, "x".repeat(129))))
                .andExpect(status().isUnauthorized());
        verify(encoder, never()).matches(any(CharSequence.class), anyString());
    }

    @Test
    void logoutRevokesAndClearsCookie() throws Exception {
        String raw = sid(login(newUser()));
        MvcResult out = mvc.perform(json(post("/session/logout"), "").cookie(new Cookie("sid", raw)))
                .andExpect(status().isNoContent()).andReturn();
        assertThat(out.getResponse().getHeader(HttpHeaders.SET_COOKIE)).startsWith("sid=;").contains("Max-Age=0");
        assertThat(store.find(SessionService.hashId(raw))).isEmpty();
        mvc.perform(get("/session/me").cookie(new Cookie("sid", raw))).andExpect(status().isUnauthorized());
    }

    @Test
    void logoutWithoutSessionStillClearsCookie() throws Exception {
        mvc.perform(json(post("/session/logout"), "")).andExpect(status().isNoContent());
    }

    @Test
    void loginReplacesTheSessionTheRequestCarried() throws Exception {
        String email = newUser();
        String planted = sid(login(email));

        MvcResult second = login(email, new Cookie("sid", planted));
        String fresh = sid(second);

        assertThat(fresh).isNotEqualTo(planted);
        assertThat(store.find(SessionService.hashId(planted))).as("old ID revoked").isEmpty();
        mvc.perform(get("/session/me").cookie(new Cookie("sid", planted))).andExpect(status().isUnauthorized());
        mvc.perform(get("/session/me").cookie(new Cookie("sid", fresh))).andExpect(status().isOk());
    }

    @Test
    void roleChangeAppliesOnTheNextRequest() throws Exception {
        String email = newUser();
        String raw = sid(login(email));
        mvc.perform(get("/session/me").cookie(new Cookie("sid", raw))).andExpect(jsonPath("$.roles[0]").value("user"));

        jdbc.sql("UPDATE users SET role = 'admin' WHERE email = CAST(:e AS citext)").param("e", email).update();

        mvc.perform(get("/session/me").cookie(new Cookie("sid", raw)))
                .andExpect(status().isOk()).andExpect(jsonPath("$.roles[0]").value("admin"));
    }

    @Test
    void expiredSessionIs401AndTheRowIsDeleted() throws Exception {
        UUID user = users.findByEmail(newUser()).orElseThrow().id();
        String raw = "expired-" + UUID.randomUUID();
        store.create(new Session(SessionService.hashId(raw), user, Instant.now().minus(1, ChronoUnit.SECONDS)));

        mvc.perform(get("/session/me").cookie(new Cookie("sid", raw))).andExpect(status().isUnauthorized());
        assertThat(store.find(SessionService.hashId(raw))).isEmpty();
    }

    @Test
    void deletedUserInvalidatesTheSession() throws Exception {
        String email = newUser();
        String raw = sid(login(email));
        jdbc.sql("DELETE FROM users WHERE email = CAST(:e AS citext)").param("e", email).update();
        mvc.perform(get("/session/me").cookie(new Cookie("sid", raw))).andExpect(status().isUnauthorized());
    }

    @Test
    void garbageCookieIs401() throws Exception {
        mvc.perform(get("/session/me").cookie(new Cookie("sid", "garbage"))).andExpect(status().isUnauthorized());
        mvc.perform(get("/session/me").cookie(new Cookie("sid", ""))).andExpect(status().isUnauthorized());
    }

    @Test
    void sessionCookieIsNotAcceptedInTokenMode() throws Exception {
        String raw = sid(login(newUser()));
        mvc.perform(get("/token/me").cookie(new Cookie("sid", raw))).andExpect(status().isUnauthorized());
    }
}
