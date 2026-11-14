package io.github.brianhuanggg.authkit.config;

import java.net.URI;
import java.net.URISyntaxException;
import java.nio.charset.StandardCharsets;
import java.util.List;
import org.springframework.boot.context.properties.ConfigurationProperties;

/**
 * Every environment variable of the Go API, bound through {@code application.yml}.
 * The compact constructor fails fast, so the app never starts with an unsafe
 * default. Empty values mean "unset", as in Go.
 */
@ConfigurationProperties("authkit")
public record AuthkitProperties(
        String addr,
        String databaseUrl,
        String jwtSecret,
        List<String> jwtPreviousSecrets,
        String jwtIssuer,
        String passwordPepper,
        List<String> passwordPreviousPeppers,
        List<String> corsOrigins,
        String logFormat,
        String cookieInsecure,
        Integer rateLimitIpBurst,
        Integer rateLimitEmailBurst) {

    public static final int DEFAULT_IP_BURST = 20;
    public static final int DEFAULT_EMAIL_BURST = 5;

    public AuthkitProperties {
        addr = blank(addr) ? ":8080" : addr;
        jwtIssuer = blank(jwtIssuer) ? "authkit" : jwtIssuer;
        passwordPepper = blank(passwordPepper) ? null : passwordPepper;
        logFormat = blank(logFormat) ? "text" : logFormat;
        if (!logFormat.equals("text") && !logFormat.equals("json")) {
            throw new IllegalArgumentException(
                    "LOG_FORMAT must be text or json (got \"" + logFormat + "\")");
        }
        cookieInsecure = cookieInsecure == null ? "" : cookieInsecure;
        jwtPreviousSecrets = clean(jwtPreviousSecrets);
        passwordPreviousPeppers = clean(passwordPreviousPeppers);
        corsOrigins = clean(corsOrigins);
        for (String origin : corsOrigins) {
            validateOrigin(origin);
        }
        rateLimitIpBurst = positive("RATE_LIMIT_IP_BURST", rateLimitIpBurst, DEFAULT_IP_BURST);
        rateLimitEmailBurst = positive("RATE_LIMIT_EMAIL_BURST", rateLimitEmailBurst, DEFAULT_EMAIL_BURST);
        if (blank(databaseUrl)) {
            throw new IllegalArgumentException("DATABASE_URL is required");
        }
        DatabaseUrl.parse(databaseUrl);
        int secretLength = jwtSecret == null ? 0 : jwtSecret.getBytes(StandardCharsets.UTF_8).length;
        if (secretLength < 32) {
            throw new IllegalArgumentException(
                    "JWT_SECRET is required and must be at least 32 bytes (got " + secretLength + ")");
        }
        for (String previous : jwtPreviousSecrets) {
            int length = previous.getBytes(StandardCharsets.UTF_8).length;
            if (length < 32) {
                throw new IllegalArgumentException(
                        "JWT_PREVIOUS_SECRETS: each secret must be at least 32 bytes (got " + length + ")");
            }
        }
    }

    public DatabaseUrl database() {
        return DatabaseUrl.parse(databaseUrl);
    }

    public boolean logJson() {
        return logFormat.equals("json");
    }

    /** {@code COOKIE_INSECURE}: only {@code 1} and {@code true} drop the Secure flag. */
    public boolean insecureCookies() {
        return cookieInsecure.equals("1") || cookieInsecure.equals("true");
    }

    /**
     * An origin is {@code scheme://host[:port]}: http or https, no path, no
     * trailing slash, no query, no user info, and never {@code *}.
     */
    static void validateOrigin(String origin) {
        URI uri;
        try {
            uri = new URI(origin);
        } catch (URISyntaxException e) {
            throw badOrigin(origin);
        }
        boolean schemeOk = "http".equals(uri.getScheme()) || "https".equals(uri.getScheme());
        boolean bare = uri.getRawPath() == null || uri.getRawPath().isEmpty();
        if (!schemeOk || uri.getHost() == null || !bare || uri.getRawQuery() != null
                || uri.getRawFragment() != null || uri.getRawUserInfo() != null || origin.endsWith("/")) {
            throw badOrigin(origin);
        }
    }

    private static IllegalArgumentException badOrigin(String origin) {
        return new IllegalArgumentException("CORS_ORIGINS: invalid origin \"" + origin
                + "\": want scheme://host[:port], e.g. http://localhost:3000");
    }

    // A limit can be raised, never silently disabled.
    private static int positive(String name, Integer value, int def) {
        if (value == null) {
            return def;
        }
        if (value <= 0) {
            throw new IllegalArgumentException(name + " must be a positive integer, got \"" + value + "\"");
        }
        return value;
    }

    private static List<String> clean(List<String> values) {
        if (values == null) {
            return List.of();
        }
        return values.stream().map(String::trim).filter(s -> !s.isEmpty()).toList();
    }

    private static boolean blank(String s) {
        return s == null || s.isBlank();
    }
}
