package io.github.brianhuanggg.authkit.token;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import com.nimbusds.jose.JWSAlgorithm;
import com.nimbusds.jose.JWSHeader;
import com.nimbusds.jose.PlainHeader;
import com.nimbusds.jose.crypto.MACSigner;
import com.nimbusds.jwt.JWTClaimsSet;
import com.nimbusds.jwt.PlainJWT;
import com.nimbusds.jwt.SignedJWT;
import io.github.brianhuanggg.authkit.auth.AuthenticatedPrincipal;
import java.nio.charset.StandardCharsets;
import java.time.Clock;
import java.time.Instant;
import java.time.ZoneOffset;
import java.util.Date;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import org.junit.jupiter.api.Test;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtException;

class JwtServiceTest {

    static final String SECRET = "0123456789abcdef0123456789abcdef";
    static final String OTHER = "fedcba9876543210fedcba9876543210";
    static final Instant T0 = Instant.parse("2026-01-01T12:00:00Z");
    static final UUID USER = UUID.fromString("11111111-1111-1111-1111-111111111111");

    /** A clock the test can move. */
    static final class MutableClock extends Clock {
        Instant now = T0;

        @Override
        public ZoneOffset getZone() {
            return ZoneOffset.UTC;
        }

        @Override
        public Clock withZone(java.time.ZoneId zone) {
            return this;
        }

        @Override
        public Instant instant() {
            return now;
        }
    }

    final MutableClock clock = new MutableClock();
    final JwtService jwt = new JwtService(SECRET, List.of(), "authkit", clock);

    private static String kidOf(String secret) {
        return JwtService.keyId(secret.getBytes(StandardCharsets.UTF_8));
    }

    private Map<String, Object> good() {
        Map<String, Object> m = new LinkedHashMap<>();
        m.put("sub", USER.toString());
        m.put("iss", "authkit");
        m.put("iat", Date.from(T0));
        m.put("exp", Date.from(T0.plusSeconds(60)));
        return m;
    }

    private static JWTClaimsSet claims(Map<String, Object> m) {
        JWTClaimsSet.Builder b = new JWTClaimsSet.Builder();
        m.forEach(b::claim);
        return b.build();
    }

    /** Signs by hand with the key's kid, so each forgery fails for its own reason. */
    private static String sign(JWSAlgorithm alg, String kid, String secret, Map<String, Object> claims)
            throws Exception {
        JWSHeader.Builder header = new JWSHeader.Builder(alg);
        if (kid != null) {
            header.keyID(kid);
        }
        SignedJWT jws = new SignedJWT(header.build(), claims(claims));
        jws.sign(new MACSigner(secret.getBytes(StandardCharsets.UTF_8)));
        return jws.serialize();
    }

    private String hs256(Map<String, Object> claims) throws Exception {
        return sign(JWSAlgorithm.HS256, kidOf(SECRET), SECRET, claims);
    }

    private boolean accepts(JwtService svc, String token) {
        try {
            svc.decoder().decode(token);
            return true;
        } catch (JwtException e) {
            return false;
        }
    }

    @Test
    void roundTrip() {
        var issued = jwt.issue(new AuthenticatedPrincipal(USER, List.of("admin", "user")));
        assertThat(issued.expiresAt()).isEqualTo(T0.plus(JwtService.TTL));

        Jwt decoded = jwt.decoder().decode(issued.value());
        assertThat(decoded.getSubject()).isEqualTo(USER.toString());
        assertThat(decoded.getClaimAsString("iss")).isEqualTo("authkit");
        assertThat(decoded.getHeaders()).containsEntry("alg", "HS256").containsEntry("kid", kidOf(SECRET));
        var auth = JwtService.toAuthentication(decoded);
        assertThat(auth.getPrincipal()).isEqualTo(new AuthenticatedPrincipal(USER, List.of("admin", "user")));
    }

    @Test
    void noRolesMeansNoRolesClaim() {
        Jwt decoded = jwt.decoder().decode(jwt.issue(new AuthenticatedPrincipal(USER, List.of())).value());
        assertThat(decoded.getClaims()).doesNotContainKey("roles");
        assertThat(JwtService.toAuthentication(decoded).getAuthorities()).isEmpty();
    }

    @Test
    void expiryHasNoLeeway() {
        String token = jwt.issue(new AuthenticatedPrincipal(USER, List.of())).value();
        clock.now = T0.plus(JwtService.TTL).minusSeconds(1);
        assertThat(accepts(jwt, token)).isTrue();
        clock.now = T0.plus(JwtService.TTL);
        assertThat(accepts(jwt, token)).isFalse();
    }

    @Test
    void rejectsForgedOrInvalid() throws Exception {
        String valid = hs256(good());
        String[] parts = valid.split("\\.");
        String longSecret = SECRET + SECRET; // HS512 signing needs a 64-byte key

        Map<String, String> cases = new LinkedHashMap<>();
        cases.put("alg=none", new PlainJWT(
                new PlainHeader.Builder().customParam("kid", kidOf(SECRET)).build(), claims(good())).serialize());
        cases.put("alg=HS512", sign(JWSAlgorithm.HS512, kidOf(SECRET), longSecret, good()));
        cases.put("wrong secret", sign(JWSAlgorithm.HS256, kidOf(SECRET), "another-secret-another-secret-123", good()));
        cases.put("wrong issuer", hs256(with(good(), "iss", "evil")));
        cases.put("missing issuer", hs256(without(good(), "iss")));
        cases.put("missing exp", hs256(without(good(), "exp")));
        cases.put("expired", hs256(with(good(), "exp", Date.from(T0.minusSeconds(60)))));
        cases.put("iat in future", hs256(with(good(), "iat", Date.from(T0.plusSeconds(3600)))));
        cases.put("missing sub", hs256(without(good(), "sub")));
        cases.put("empty sub", hs256(with(good(), "sub", "")));
        cases.put("sub not a uuid", hs256(with(good(), "sub", "admin")));
        cases.put("roles not strings", hs256(with(good(), "roles", List.of(1, 2))));
        cases.put("tampered signature", parts[0] + "." + parts[1] + "." + "A".repeat(parts[2].length()));
        cases.put("tampered claims", parts[0] + "." + "eyJzdWIiOiJhZG1pbiJ9" + "." + parts[2]);
        cases.put("unknown kid", sign(JWSAlgorithm.HS256, "nope", SECRET, good()));
        cases.put("no kid", sign(JWSAlgorithm.HS256, null, SECRET, good()));
        cases.put("garbage", "not-a-jwt");
        cases.put("empty", "");

        assertThat(accepts(jwt, valid)).as("control: the untouched token").isTrue();
        cases.forEach((name, token) -> assertThat(accepts(jwt, token)).as(name).isFalse());
    }

    @Test
    void keyRotation() {
        String oldToken = jwt.issue(new AuthenticatedPrincipal(USER, List.of())).value();

        JwtService rotated = new JwtService(OTHER, List.of(SECRET), "authkit", clock);
        assertThat(accepts(rotated, oldToken)).as("token from previous key").isTrue();

        String newToken = rotated.issue(new AuthenticatedPrincipal(USER, List.of())).value();
        assertThat(accepts(jwt, newToken)).as("old instance, key it does not know").isFalse();

        JwtService dropped = new JwtService(OTHER, List.of(), "authkit", clock);
        assertThat(accepts(dropped, oldToken)).as("previous key dropped").isFalse();
    }

    @Test
    void rejectsBadKeys() {
        assertThatThrownBy(() -> new JwtService("short", List.of(), "authkit", clock))
                .isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> new JwtService(SECRET, List.of("short"), "authkit", clock))
                .isInstanceOf(IllegalArgumentException.class);
        assertThatThrownBy(() -> new JwtService(SECRET, List.of(SECRET), "authkit", clock))
                .isInstanceOf(IllegalArgumentException.class).hasMessageContaining("duplicate");
        assertThatThrownBy(() -> new JwtService(SECRET, List.of(), "", clock))
                .isInstanceOf(IllegalArgumentException.class);
    }

    private static Map<String, Object> with(Map<String, Object> m, String k, Object v) {
        m.put(k, v);
        return m;
    }

    private static Map<String, Object> without(Map<String, Object> m, String k) {
        m.remove(k);
        return m;
    }
}
