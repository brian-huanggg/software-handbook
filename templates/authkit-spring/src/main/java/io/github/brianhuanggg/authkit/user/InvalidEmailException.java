package io.github.brianhuanggg.authkit.user;

public class InvalidEmailException extends RuntimeException {

    public InvalidEmailException() {
        super("invalid email");
    }
}
