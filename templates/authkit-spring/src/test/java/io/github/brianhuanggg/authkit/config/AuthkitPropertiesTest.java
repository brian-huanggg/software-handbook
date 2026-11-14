package io.github.brianhuanggg.authkit.config;

import static org.assertj.core.api.Assertions.assertThat;

import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;
import org.springframework.boot.context.properties.EnableConfigurationProperties;
import org.springframework.boot.test.context.runner.ApplicationContextRunner;
import org.springframework.core.NestedExceptionUtils;

class AuthkitPropertiesTest {

    private static final String SECRET = "k".repeat(32);

    @EnableConfigurationProperties(AuthkitProperties.class)
    static class Props {
    }

    private final ApplicationContextRunner runner = new ApplicationContextRunner().withUserConfiguration(Props.class);

    /** Properties as the env vars map to them, with only the required ones set. */
    private ApplicationContextRunner with(String... extra) {
        String[] base = {"authkit.database-url=postgres://u:p@db:5432/authkit", "authkit.jwt-secret=" + SECRET};
        String[] all = new String[base.length + extra.length];
        System.arraycopy(base, 0, all, 0, base.length);
        System.arraycopy(extra, 0, all, base.length, extra.length);
        return runner.withPropertyValues(all);
    }

    private void assertRejected(ApplicationContextRunner r, String messagePart) {
        r.run(ctx -> {
            assertThat(ctx).hasFailed();
            Throwable cause = NestedExceptionUtils.getMostSpecificCause(ctx.getStartupFailure());
            assertThat(cause).hasMessageContaining(messagePart);
        });
    }

    @Test
    void defaults() {
        with().run(ctx -> {
            AuthkitProperties p = ctx.getBean(AuthkitProperties.class);
            assertThat(p.addr()).isEqualTo(":8080");
            assertThat(p.jwtIssuer()).isEqualTo("authkit");
            assertThat(p.passwordPepper()).isNull();
            assertThat(p.passwordPreviousPeppers()).isEmpty();
            assertThat(p.jwtPreviousSecrets()).isEmpty();
            assertThat(p.corsOrigins()).isEmpty();
            assertThat(p.logJson()).isFalse();
            assertThat(p.insecureCookies()).isFalse();
            assertThat(p.rateLimitIpBurst()).isEqualTo(20);
            assertThat(p.rateLimitEmailBurst()).isEqualTo(5);
        });
    }

    @Test
    void emptyValuesMeanUnset() {
        // The compose file passes unset variables through as empty strings.
        with("authkit.addr=", "authkit.jwt-issuer=", "authkit.password-pepper=", "authkit.cors-origins=",
                "authkit.log-format=", "authkit.cookie-insecure=", "authkit.rate-limit-ip-burst=",
                "authkit.rate-limit-email-burst=", "authkit.jwt-previous-secrets=").run(ctx -> {
            assertThat(ctx).hasNotFailed();
            AuthkitProperties p = ctx.getBean(AuthkitProperties.class);
            assertThat(p.addr()).isEqualTo(":8080");
            assertThat(p.rateLimitIpBurst()).isEqualTo(20);
            assertThat(p.rateLimitEmailBurst()).isEqualTo(5);
        });
    }

    @Test
    void overrides() {
        with("authkit.addr=:9", "authkit.jwt-issuer=me", "authkit.cookie-insecure=1",
                "authkit.password-pepper=pp", "authkit.password-previous-peppers= old1 , old2 ,",
                "authkit.jwt-previous-secrets=" + "a".repeat(32) + "," + "b".repeat(33),
                "authkit.cors-origins=http://localhost:3000, https://app.example.com:8443",
                "authkit.log-format=json", "authkit.rate-limit-ip-burst=500",
                "authkit.rate-limit-email-burst=50").run(ctx -> {
            AuthkitProperties p = ctx.getBean(AuthkitProperties.class);
            assertThat(p.addr()).isEqualTo(":9");
            assertThat(p.jwtIssuer()).isEqualTo("me");
            assertThat(p.insecureCookies()).isTrue();
            assertThat(p.passwordPepper()).isEqualTo("pp");
            assertThat(p.passwordPreviousPeppers()).containsExactly("old1", "old2");
            assertThat(p.jwtPreviousSecrets()).hasSize(2);
            assertThat(p.corsOrigins()).containsExactly("http://localhost:3000", "https://app.example.com:8443");
            assertThat(p.logJson()).isTrue();
            assertThat(p.rateLimitIpBurst()).isEqualTo(500);
            assertThat(p.rateLimitEmailBurst()).isEqualTo(50);
        });
    }

    @Test
    void cookieInsecureOnlyOneOrTrue() {
        for (Map.Entry<String, Boolean> e : Map.of("1", true, "true", true, "0", false, "yes", false, "", false)
                .entrySet()) {
            with("authkit.cookie-insecure=" + e.getKey()).run(ctx ->
                    assertThat(ctx.getBean(AuthkitProperties.class).insecureCookies())
                            .as("COOKIE_INSECURE=%s", e.getKey()).isEqualTo(e.getValue()));
        }
    }

    @Test
    void rejectsMissingDatabaseUrl() {
        assertRejected(runner.withPropertyValues("authkit.jwt-secret=" + SECRET), "DATABASE_URL is required");
    }

    @ParameterizedTest
    @ValueSource(strings = {"x", "mysql://u@h/db", "postgres:///db", "http://h/db"})
    void rejectsMalformedDatabaseUrl(String url) {
        assertRejected(with("authkit.database-url=" + url), "DATABASE_URL must look like");
    }

    @Test
    void rejectsMissingJwtSecret() {
        assertRejected(runner.withPropertyValues("authkit.database-url=postgres://u@h/db"), "JWT_SECRET");
    }

    @Test
    void rejectsShortJwtSecret() {
        assertRejected(with("authkit.jwt-secret=short"), "at least 32 bytes (got 5)");
    }

    @Test
    void jwtSecretLengthIsInBytesNotCharacters() {
        // 11 characters, 33 bytes in UTF-8: long enough.
        with("authkit.jwt-secret=" + "€".repeat(11)).run(ctx -> assertThat(ctx).hasNotFailed());
        // 10 characters, 30 bytes: too short although more than 8 characters.
        assertRejected(with("authkit.jwt-secret=" + "€".repeat(10)), "(got 30)");
    }

    @Test
    void rejectsShortPreviousJwtSecret() {
        assertRejected(with("authkit.jwt-previous-secrets=" + SECRET + ",short"), "JWT_PREVIOUS_SECRETS");
    }

    @ParameterizedTest
    @ValueSource(strings = {"*", "http://localhost:3000/", "localhost:3000", "http://a,*", "ftp://a",
            "http://a/path", "http://a?x=1", "http://u@a", "http://a#f"})
    void rejectsBadCorsOrigin(String origins) {
        assertRejected(with("authkit.cors-origins=" + origins), "CORS_ORIGINS");
    }

    @ParameterizedTest
    @ValueSource(strings = {"0", "-1", "lots"})
    void rejectsBadIpBurst(String v) {
        runner.withPropertyValues("authkit.database-url=postgres://u@h/db", "authkit.jwt-secret=" + SECRET,
                "authkit.rate-limit-ip-burst=" + v).run(ctx -> assertThat(ctx).hasFailed());
    }

    @ParameterizedTest
    @ValueSource(strings = {"0", "-1", "lots"})
    void rejectsBadEmailBurst(String v) {
        runner.withPropertyValues("authkit.database-url=postgres://u@h/db", "authkit.jwt-secret=" + SECRET,
                "authkit.rate-limit-email-burst=" + v).run(ctx -> assertThat(ctx).hasFailed());
    }

    @Test
    void rejectsZeroBurstWithName() {
        assertRejected(with("authkit.rate-limit-ip-burst=0"), "RATE_LIMIT_IP_BURST must be a positive integer");
        assertRejected(with("authkit.rate-limit-email-burst=0"), "RATE_LIMIT_EMAIL_BURST must be a positive integer");
    }

    @Test
    void rejectsUnknownLogFormat() {
        assertRejected(with("authkit.log-format=xml"), "LOG_FORMAT must be text or json");
    }

    @Test
    void databaseUrlSplitsIntoJdbcParts() {
        DatabaseUrl db = DatabaseUrl.parse("postgres://authkit:p%40ss@db:5432/authkit?sslmode=disable");
        assertThat(db.jdbcUrl()).isEqualTo("jdbc:postgresql://db:5432/authkit?sslmode=disable");
        assertThat(db.username()).isEqualTo("authkit");
        assertThat(db.password()).isEqualTo("p@ss");
        assertThat(DatabaseUrl.parse("postgresql://h/db").jdbcUrl()).isEqualTo("jdbc:postgresql://h/db");
        assertThat(DatabaseUrl.parse("jdbc:postgresql://h/db").username()).isNull();
    }

    @Test
    void databaseUrlErrorHidesThePassword() {
        assertThat(List.of("mysql://u:topsecret@h/db")).allSatisfy(url -> {
            try {
                DatabaseUrl.parse(url);
            } catch (IllegalArgumentException e) {
                assertThat(e.getMessage()).doesNotContain("topsecret");
            }
        });
    }
}
