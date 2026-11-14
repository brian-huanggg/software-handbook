package io.github.brianhuanggg.authkit.web;

/** Request body of {@code /register} and both logins. */
public record Credentials(String email, String password) {
}
