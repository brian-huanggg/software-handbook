package io.github.brianhuanggg.authkit.session;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

import io.github.brianhuanggg.authkit.PostgresTestBase;
import io.github.brianhuanggg.authkit.password.PepperedArgon2Encoder;
import io.github.brianhuanggg.authkit.user.UserRepository;
import java.util.List;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.security.crypto.password.PasswordEncoder;
import org.springframework.test.context.TestPropertySource;
import org.springframework.test.web.servlet.MockMvc;

/** A hash made before a pepper was set is upgraded at the next successful login. */
@AutoConfigureMockMvc
@TestPropertySource(properties = "authkit.password-pepper=pepper-one")
class PasswordUpgradeOnLoginTest extends PostgresTestBase {

    @Autowired
    MockMvc mvc;
    @Autowired
    UserRepository users;
    @Autowired
    PasswordEncoder encoder;

    private String login(String email, String password) throws Exception {
        return mvc.perform(post("/session/login").contentType(MediaType.APPLICATION_JSON)
                        .header("X-Requested-With", "XMLHttpRequest")
                        .content("{\"email\":\"%s\",\"password\":\"%s\"}".formatted(email, password)))
                .andReturn().getResponse().getStatus() + "";
    }

    @Test
    void oldHashIsUpgradedOnLogin() throws Exception {
        String email = "u-" + UUID.randomUUID() + "@example.com";
        String unpeppered = new PepperedArgon2Encoder(null, List.of(), 1).encode("correct horse");
        users.create(email, unpeppered);
        assertThat(encoder.upgradeEncoding(unpeppered)).isTrue();

        assertThat(login(email, "wrong password")).isEqualTo("401");
        assertThat(users.findByEmail(email).orElseThrow().passwordHash()).as("no upgrade on failure")
                .isEqualTo(unpeppered);

        assertThat(login(email, "correct horse")).isEqualTo("200");
        String upgraded = users.findByEmail(email).orElseThrow().passwordHash();
        assertThat(upgraded).isNotEqualTo(unpeppered).contains(",k=");
        assertThat(encoder.upgradeEncoding(upgraded)).isFalse();
        assertThat(encoder.matches("correct horse", upgraded)).isTrue();

        assertThat(login(email, "correct horse")).as("still works after the upgrade").isEqualTo("200");
    }
}
