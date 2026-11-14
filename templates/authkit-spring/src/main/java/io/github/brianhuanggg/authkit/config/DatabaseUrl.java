package io.github.brianhuanggg.authkit.config;

import java.net.URI;
import java.net.URISyntaxException;

/**
 * A {@code postgres://user:pass@host:port/db?sslmode=disable} URL, the format
 * the Go API uses for {@code DATABASE_URL}, split into what JDBC needs.
 * A ready-made {@code jdbc:postgresql:} URL is passed through unchanged.
 */
public record DatabaseUrl(String jdbcUrl, String username, String password) {

    public static DatabaseUrl parse(String value) {
        if (value.startsWith("jdbc:postgresql:")) {
            return new DatabaseUrl(value, null, null);
        }
        URI uri;
        try {
            uri = new URI(value);
        } catch (URISyntaxException e) {
            throw invalid();
        }
        String scheme = uri.getScheme();
        if (!"postgres".equals(scheme) && !"postgresql".equals(scheme)) {
            throw invalid();
        }
        if (uri.getHost() == null) {
            throw invalid();
        }
        String user = null;
        String password = null;
        String info = uri.getUserInfo();
        if (info != null) {
            int colon = info.indexOf(':');
            user = colon < 0 ? info : info.substring(0, colon);
            password = colon < 0 ? null : info.substring(colon + 1);
        }
        StringBuilder jdbc = new StringBuilder("jdbc:postgresql://").append(uri.getHost());
        if (uri.getPort() >= 0) {
            jdbc.append(':').append(uri.getPort());
        }
        jdbc.append(uri.getRawPath() == null ? "" : uri.getRawPath());
        if (uri.getRawQuery() != null) {
            jdbc.append('?').append(uri.getRawQuery());
        }
        return new DatabaseUrl(jdbc.toString(), user, password);
    }

    // The message never echoes the value: it can contain a password.
    private static IllegalArgumentException invalid() {
        return new IllegalArgumentException(
                "DATABASE_URL must look like postgres://user:pass@host:5432/db");
    }
}
