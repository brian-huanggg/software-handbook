package io.github.brianhuanggg.authkit.session;

public class DuplicateSessionException extends RuntimeException {

    public DuplicateSessionException() {
        super("session: id hash already exists");
    }
}
