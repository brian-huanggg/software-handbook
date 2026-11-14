package io.github.brianhuanggg.authkit.password;

/**
 * Every hashing slot stayed taken for the whole wait: the server is
 * overloaded, not the request wrong. Answered as 503 with {@code Retry-After}.
 */
public class PasswordHashingBusyException extends RuntimeException {

    public PasswordHashingBusyException() {
        super("password: all hashing slots busy");
    }
}
