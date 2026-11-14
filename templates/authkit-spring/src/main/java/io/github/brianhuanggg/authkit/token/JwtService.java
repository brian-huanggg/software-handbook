package io.github.brianhuanggg.authkit.token;

import com.nimbusds.jose.JWSAlgorithm;
import com.nimbusds.jose.jwk.JWK;
import com.nimbusds.jose.jwk.JWKSet;
import com.nimbusds.jose.jwk.OctetSequenceKey;
import com.nimbusds.jose.jwk.source.ImmutableJWKSet;
import com.nimbusds.jose.proc.JWSVerificationKeySelector;
import com.nimbusds.jose.proc.SecurityContext;
import com.nimbusds.jwt.proc.DefaultJWTProcessor;
import io.github.brianhuanggg.authkit.auth.AuthenticatedPrincipal;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Clock;
import java.time.Duration;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.ArrayList;
import java.util.HexFormat;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import org.springframework.security.authentication.AbstractAuthenticationToken;
import org.springframework.security.oauth2.core.DelegatingOAuth2TokenValidator;
import org.springframework.security.oauth2.core.OAuth2Error;
import org.springframework.security.oauth2.core.OAuth2TokenValidatorResult;
import org.springframework.security.oauth2.jose.jws.MacAlgorithm;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.security.oauth2.jwt.JwtClaimsSet;
import org.springframework.security.oauth2.jwt.JwtDecoder;
import org.springframework.security.oauth2.jwt.JwtEncoderParameters;
import org.springframework.security.oauth2.jwt.JwtIssuerValidator;
import org.springframework.security.oauth2.jwt.JwsHeader;
import org.springframework.security.oauth2.jwt.JwtTimestampValidator;
import org.springframework.security.oauth2.jwt.NimbusJwtDecoder;
import org.springframework.security.oauth2.jwt.NimbusJwtEncoder;

/**
 * Issues and verifies access JWTs, same format as the Go API: HS256, claims
 * {@code sub iss iat exp roles}, 15 minute lifetime, and a {@code kid} header
 * derived from the key so rotation needs no extra configuration.
 *
 * <p>Only the current secret signs. Previous secrets still verify. Tokens
 * from this class verify in the Go API and the other way round when both share
 * the same secrets.
 */
public final class JwtService {

    public static final Duration TTL = Duration.ofMinutes(15);

    private static final int MIN_SECRET_BYTES = 32;
    private static final String ROLES = "roles";

    private final Clock clock;
    private final String issuer;
    private final String signingKid;
    private final NimbusJwtEncoder encoder;
    private final NimbusJwtDecoder decoder;

    public JwtService(String secret, List<String> previousSecrets, String issuer, Clock clock) {
        if (issuer == null || issuer.isEmpty()) {
            throw new IllegalArgumentException("jwt: issuer is required");
        }
        this.clock = clock;
        this.issuer = issuer;

        // kid -> key. The signing key comes first; duplicates are a config mistake.
        Map<String, OctetSequenceKey> keys = new LinkedHashMap<>();
        List<String> all = new ArrayList<>();
        all.add(secret);
        all.addAll(previousSecrets);
        for (int i = 0; i < all.size(); i++) {
            byte[] key = all.get(i).getBytes(StandardCharsets.UTF_8);
            if (key.length < MIN_SECRET_BYTES) {
                throw new IllegalArgumentException(
                        "jwt: secret %d must be at least %d bytes".formatted(i, MIN_SECRET_BYTES));
            }
            String kid = keyId(key);
            if (keys.containsKey(kid)) {
                throw new IllegalArgumentException("jwt: secret %d is a duplicate".formatted(i));
            }
            keys.put(kid, new OctetSequenceKey.Builder(key).keyID(kid).build());
        }
        this.signingKid = keys.keySet().iterator().next();

        this.encoder = new NimbusJwtEncoder(
                new ImmutableJWKSet<>(new JWKSet(List.<JWK>of(keys.get(signingKid)))));
        this.decoder = buildDecoder(new JWKSet(List.<JWK>copyOf(keys.values())));
    }

    /** A signed access token and the moment it stops being valid. */
    public record IssuedToken(String value, Instant expiresAt) {
    }

    public IssuedToken issue(AuthenticatedPrincipal principal) {
        // Whole seconds, like the numeric dates in the token itself.
        Instant now = clock.instant().truncatedTo(ChronoUnit.SECONDS);
        Instant exp = now.plus(TTL);
        JwtClaimsSet.Builder claims = JwtClaimsSet.builder()
                .subject(principal.userId().toString())
                .issuer(issuer)
                .issuedAt(now)
                .expiresAt(exp);
        if (!principal.roles().isEmpty()) {
            claims.claim(ROLES, principal.roles());
        }
        JwsHeader header = JwsHeader.with(MacAlgorithm.HS256).keyId(signingKid).build();
        String token = encoder.encode(JwtEncoderParameters.from(header, claims.build())).getTokenValue();
        return new IssuedToken(token, exp);
    }

    /** The verifier used by {@code oauth2ResourceServer}. Every failure is a {@code JwtException}. */
    public JwtDecoder decoder() {
        return decoder;
    }

    /** Converts a verified token to the same principal the session mode uses. */
    public static AbstractAuthenticationToken toAuthentication(Jwt jwt) {
        List<String> roles = jwt.getClaimAsStringList(ROLES);
        return new AuthenticatedPrincipal(UUID.fromString(jwt.getSubject()), roles == null ? List.of() : roles)
                .toAuthentication();
    }

    private NimbusJwtDecoder buildDecoder(JWKSet keys) {
        DefaultJWTProcessor<SecurityContext> processor = new DefaultJWTProcessor<>();
        // HS256 is the only accepted algorithm, whatever the token's own header says.
        // That blocks alg=none and algorithm confusion. The key comes from the kid header.
        processor.setJWSKeySelector(new JWSVerificationKeySelector<>(JWSAlgorithm.HS256, new ImmutableJWKSet<>(keys)));
        // Nimbus would accept 60 seconds past expiry. Our own validators decide instead.
        processor.setJWTClaimsSetVerifier((claims, context) -> {
        });

        JwtTimestampValidator timestamps = new JwtTimestampValidator(Duration.ZERO);
        timestamps.setClock(clock);
        NimbusJwtDecoder decoder = new NimbusJwtDecoder(processor);
        decoder.setJwtValidator(new DelegatingOAuth2TokenValidator<>(
                new JwtIssuerValidator(issuer), timestamps, this::requiredClaims));
        return decoder;
    }

    /**
     * What the other validators do not check: a {@code kid} (a token without
     * one is rejected, not tried against every key), {@code exp}, a UUID
     * {@code sub}, {@code iat} not in the future, and {@code roles} as a list of strings.
     */
    private OAuth2TokenValidatorResult requiredClaims(Jwt jwt) {
        Instant now = clock.instant();
        boolean ok = jwt.getHeaders().get("kid") instanceof String kid && !kid.isEmpty()
                // Valid strictly before exp, like Go. Spring's own check still accepts the exp second itself.
                && jwt.getExpiresAt() != null && jwt.getExpiresAt().isAfter(now)
                && (jwt.getIssuedAt() == null || !jwt.getIssuedAt().isAfter(now))
                && isUuid(jwt.getSubject())
                && rolesOk(jwt.getClaim(ROLES));
        return ok ? OAuth2TokenValidatorResult.success()
                : OAuth2TokenValidatorResult.failure(new OAuth2Error("invalid_token"));
    }

    private static boolean isUuid(String s) {
        if (s == null || s.isEmpty()) {
            return false;
        }
        try {
            UUID.fromString(s);
            return true;
        } catch (IllegalArgumentException e) {
            return false;
        }
    }

    private static boolean rolesOk(Object roles) {
        return roles == null || (roles instanceof List<?> list && list.stream().allMatch(String.class::isInstance));
    }

    /**
     * A truncated hash of a high-entropy secret, so it reveals nothing usable
     * about the key. Same derivation as the Go API.
     */
    static String keyId(byte[] key) {
        try {
            MessageDigest sha = MessageDigest.getInstance("SHA-256");
            sha.update("authkit-kid:".getBytes(StandardCharsets.UTF_8));
            sha.update(key);
            return HexFormat.of().formatHex(sha.digest(), 0, 8);
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException(e);
        }
    }
}
