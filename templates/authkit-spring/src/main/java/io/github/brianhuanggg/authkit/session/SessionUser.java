package io.github.brianhuanggg.authkit.session;

/**
 * A stored session together with its user's current role, read in one lookup.
 *
 * @param role read from {@code users} at lookup time, so a role change applies on the next request
 */
public record SessionUser(Session session, String role) {
}
