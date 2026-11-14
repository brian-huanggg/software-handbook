package io.github.brianhuanggg.authkit.user;

public class EmailTakenException extends RuntimeException {

    public EmailTakenException() {
        super("email already registered");
    }
}
