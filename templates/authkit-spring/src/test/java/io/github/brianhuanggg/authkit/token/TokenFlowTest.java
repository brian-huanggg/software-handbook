package io.github.brianhuanggg.authkit.token;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import jakarta.servlet.http.Cookie;
import java.util.List;
import java.util.UUID;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.jdbc.core.simple.JdbcClient;
import org.springframework.test.web.servlet.MockMvc;
import org.springframework.test.web.servlet.MvcResult;
import org.springframework.test.web.servlet.request.MockHttpServletRequestBuilder;

/** {@code /token/login}, {@code /token/refresh} and {@code /token/logout} against real Postgres. */
@AutoConfigureMockMvc
class TokenFlowTest extends PostgresTestBase {

    static final String PASSWORD = "correct horse battery";

    @Autowired
    MockMvc mvc;
    @Autowired
    JdbcClient jdbc;

    private static MockHttpServletRequestBuilder csrf(MockHttpServletRequestBuilder b) {
        return b.header("X-Requested-With", "XMLHttpRequest");
    }

    private String newUser() throws Exception {
        String email = "u-" + UUID.randomUUID() + "@example.com";
        mvc.perform(csrf(post("/register")).contentType(MediaType.APPLICATION_JSON)
                .content("{\"email\":\"%s\",\"password\":\"%s\"}".formatted(email, PASSWORD)))
                .andExpect(status().isAccepted());
        return email;
    }

    private MvcResult login(String email) throws Exception {
        return mvc.perform(csrf(post("/token/login")).contentType(MediaType.APPLICATION_JSON)
                .content("{\"email\":\"%s\",\"password\":\"%s\"}".formatted(email, PASSWORD)))
                .andExpect(status().isOk()).andReturn();
    }

    private static String setCookie(MvcResult r) {
        return r.getResponse().getHeaders(HttpHeaders.SET_COOKIE).stream()
                .filter(h -> h.startsWith("refresh_token=")).findFirst().orElseThrow();
    }

    private static String refreshValue(MvcResult r) {
        String h = setCookie(r);
        return h.substring("refresh_token=".length(), h.indexOf(';'));
    }

    private MvcResult refresh(String value) throws Exception {
        var req = csrf(post("/token/refresh"));
        if (value != null) {
            req = req.cookie(new Cookie("refresh_token", value));
        }
        return mvc.perform(req).andReturn();
    }

    private long rowsFor(String email) {
        return jdbc.sql("""
                SELECT count(*) FROM refresh_tokens t JOIN users u ON u.id = t.user_id WHERE u.email = :e""")
                .param("e", email).query(Long.class).single();
    }

    @Test
    void loginIssuesAccessTokenAndCookie() throws Exception {
        String email = newUser();
        MvcResult r = login(email);

        String cookie = setCookie(r);
        assertThat(cookie).contains("HttpOnly", "Secure", "SameSite=Strict", "Path=/token", "Max-Age=604800");
        assertThat(r.getResponse().getContentAsString()).contains("\"token_type\":\"Bearer\"", "\"expires_in\":900");

        String access = com.jayway.jsonpath.JsonPath.read(r.getResponse().getContentAsString(), "$.access_token");
        mvc.perform(get("/token/me").header("Authorization", "Bearer " + access))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.roles[0]").value("user"));
    }

    @Test
    void badCredentialsAre401WithoutCookie() throws Exception {
        String email = newUser();
        MvcResult r = mvc.perform(csrf(post("/token/login")).contentType(MediaType.APPLICATION_JSON)
                .content("{\"email\":\"%s\",\"password\":\"wrong wrong wrong\"}".formatted(email)))
                .andExpect(status().isUnauthorized()).andReturn();
        assertThat(r.getResponse().getHeaders(HttpHeaders.SET_COOKIE)).isEmpty();
        assertThat(rowsFor(email)).isZero();
    }

    @Test
    void refreshRotatesTheCookie() throws Exception {
        String email = newUser();
        String first = refreshValue(login(email));

        MvcResult r = refresh(first);
        assertThat(r.getResponse().getStatus()).isEqualTo(200);
        String second = refreshValue(r);
        assertThat(second).isNotEqualTo(first);
        assertThat(setCookie(r)).contains("HttpOnly", "SameSite=Strict", "Path=/token");
        assertThat(r.getResponse().getContentAsString()).contains("access_token");
        assertThat(rowsFor(email)).isEqualTo(2);
    }

    @Test
    void reuseRevokesTheFamilyAndClearsTheCookie() throws Exception {
        String email = newUser();
        String first = refreshValue(login(email));
        String second = refreshValue(refresh(first));

        MvcResult reused = refresh(first);
        assertThat(reused.getResponse().getStatus()).isEqualTo(401);
        assertThat(setCookie(reused)).contains("Max-Age=0");
        assertThat(reused.getResponse().getContentAsString()).contains("\"error\":\"unauthenticated\"");
        assertThat(rowsFor(email)).as("family deleted").isZero();
        assertThat(refresh(second).getResponse().getStatus()).as("legitimate child").isEqualTo(401);
    }

    @Test
    void missingCookieIs401AndClearsNothing() throws Exception {
        MvcResult r = refresh(null);
        assertThat(r.getResponse().getStatus()).isEqualTo(401);
        assertThat(r.getResponse().getHeaders(HttpHeaders.SET_COOKIE)).isEmpty();
    }

    @Test
    void unknownCookieIs401AndClearsIt() throws Exception {
        MvcResult r = refresh("not-a-token");
        assertThat(r.getResponse().getStatus()).isEqualTo(401);
        assertThat(setCookie(r)).contains("Max-Age=0");
    }

    @Test
    void roleChangeAppliesOnRefresh() throws Exception {
        String email = newUser();
        String first = refreshValue(login(email));
        jdbc.sql("UPDATE users SET role = 'admin' WHERE email = :e").param("e", email).update();

        MvcResult r = refresh(first);
        String access = com.jayway.jsonpath.JsonPath.read(r.getResponse().getContentAsString(), "$.access_token");
        mvc.perform(get("/token/me").header("Authorization", "Bearer " + access))
                .andExpect(jsonPath("$.roles[0]").value("admin"));
    }

    @Test
    void logoutRevokesTheFamilyAndIsIdempotent() throws Exception {
        String email = newUser();
        String first = refreshValue(refreshOnce(email));

        MvcResult out = mvc.perform(csrf(post("/token/logout")).cookie(new Cookie("refresh_token", first)))
                .andExpect(status().isNoContent())
                .andExpect(header().string("Cache-Control", "no-store")).andReturn();
        assertThat(setCookie(out)).contains("Max-Age=0", "Path=/token");
        assertThat(rowsFor(email)).isZero();
        assertThat(refresh(first).getResponse().getStatus()).isEqualTo(401);

        mvc.perform(csrf(post("/token/logout"))).andExpect(status().isNoContent());
    }

    private MvcResult refreshOnce(String email) throws Exception {
        return refresh(refreshValue(login(email)));
    }

    /** Two refreshes with the same token at once: one wins, the loser revokes the family. */
    @Test
    void parallelRefreshesEndWithTheFamilyRevoked() throws Exception {
        for (int round = 0; round < 5; round++) {
            String email = newUser();
            String first = refreshValue(login(email));

            CountDownLatch go = new CountDownLatch(1);
            ExecutorService pool = Executors.newFixedThreadPool(2);
            try {
                List<Future<MvcResult>> both = List.of(
                        pool.submit(() -> {
                            go.await();
                            return refresh(first);
                        }),
                        pool.submit(() -> {
                            go.await();
                            return refresh(first);
                        }));
                go.countDown();
                MvcResult a = both.get(0).get();
                MvcResult b = both.get(1).get();

                int ok = (a.getResponse().getStatus() == 200 ? 1 : 0) + (b.getResponse().getStatus() == 200 ? 1 : 0);
                assertThat(ok).as("successful refreshes").isEqualTo(1);
                assertThat(a.getResponse().getStatus() + b.getResponse().getStatus()).isEqualTo(200 + 401);

                MvcResult winner = a.getResponse().getStatus() == 200 ? a : b;
                assertThat(rowsFor(email)).as("family revoked").isZero();
                assertThat(refresh(refreshValue(winner)).getResponse().getStatus())
                        .as("winner's new token").isEqualTo(401);
            } finally {
                pool.shutdownNow();
            }
        }
    }
}
