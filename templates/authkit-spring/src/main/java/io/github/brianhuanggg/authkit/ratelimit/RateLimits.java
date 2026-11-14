package io.github.brianhuanggg.authkit.ratelimit;

/**
 * The two brute-force limits. {@code perIp} applies to {@code /register} and
 * both logins; {@code perEmail} applies to both logins.
 */
public record RateLimits(Limiter perIp, Limiter perEmail) {
}
