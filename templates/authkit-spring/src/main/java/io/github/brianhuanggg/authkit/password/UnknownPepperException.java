package io.github.brianhuanggg.authkit.password;

/**
 * A stored hash names a pepper this encoder was not given. A configuration
 * error (a dropped pepper), not a wrong password.
 */
public class UnknownPepperException extends IllegalStateException {

    public UnknownPepperException() {
        super("password: hash uses an unknown pepper");
    }
}
