package io.github.brianhuanggg.authkit.token;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.header;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import io.github.brianhuanggg.authkit.auth.AuthenticatedPrincipal;
import java.util.List;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.test.web.servlet.MockMvc;

/** The {@code /token/**} chain end to end: real filters, real decoder. */
@AutoConfigureMockMvc
class TokenChainTest extends PostgresTestBase {

    static final String CHALLENGE = "Bearer realm=\"authkit\"";

    @Autowired
    MockMvc mvc;

    @Autowired
    JwtService jwt;

    @Test
    void validTokenReachesMe() throws Exception {
        UUID id = UUID.randomUUID();
        String token = jwt.issue(new AuthenticatedPrincipal(id, List.of("user"))).value();

        mvc.perform(get("/token/me").header("Authorization", "Bearer " + token))
                .andExpect(status().isOk())
                .andExpect(header().string("Cache-Control", "no-store"))
                .andExpect(jsonPath("$.user_id").value(id.toString()))
                .andExpect(jsonPath("$.roles[0]").value("user"));
    }

    @Test
    void schemeIsCaseInsensitive() throws Exception {
        String token = jwt.issue(new AuthenticatedPrincipal(UUID.randomUUID(), List.of())).value();
        mvc.perform(get("/token/me").header("Authorization", "bearer " + token))
                .andExpect(status().isOk());
    }

    @Test
    void everyFailureIsTheSame401() throws Exception {
        String token = jwt.issue(new AuthenticatedPrincipal(UUID.randomUUID(), List.of())).value();
        String[] headers = {
                null, // no header
                "Basic dTE6cHc=",
                "Bearer ",
                token, // no scheme
                "Bearer nope",
                "Bearer " + token + " junk",
                "Bearer " + token.substring(0, token.length() - 4) + "AAAA", // bad signature
        };
        for (String h : headers) {
            var req = get("/token/me");
            if (h != null) {
                req = req.header("Authorization", h);
            }
            mvc.perform(req)
                    .andExpect(status().isUnauthorized())
                    .andExpect(header().string("WWW-Authenticate", CHALLENGE))
                    .andExpect(header().string("Cache-Control", "no-store"))
                    .andExpect(jsonPath("$.error").value("unauthenticated"));
        }
    }

    @Test
    void sessionCookieIsNotAcceptedOnTokenChain() throws Exception {
        mvc.perform(get("/token/me").cookie(new jakarta.servlet.http.Cookie("sid", "whatever")))
                .andExpect(status().isUnauthorized());
    }
}
