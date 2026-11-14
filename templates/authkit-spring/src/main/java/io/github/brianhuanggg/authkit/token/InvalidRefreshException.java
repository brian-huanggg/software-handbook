package io.github.brianhuanggg.authkit.token;

/** Unknown, malformed, expired or revoked refresh token. The cause is deliberately not exposed. */
public class InvalidRefreshException extends RuntimeException {

    public InvalidRefreshException() {
        super("invalid refresh token");
    }

    protected InvalidRefreshException(String message) {
        super(message);
    }
}
