package io.github.brianhuanggg.authkit.token;

/** A refresh token with this ID hash already exists. */
public class DuplicateRefreshException extends RuntimeException {

    public DuplicateRefreshException() {
        super("refresh token id hash already exists");
    }
}
