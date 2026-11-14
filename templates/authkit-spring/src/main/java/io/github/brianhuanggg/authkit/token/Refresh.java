package io.github.brianhuanggg.authkit.token;

import java.time.Instant;
import java.util.UUID;

/**
 * A stored refresh token. {@code idHash} is the SHA-256 of the raw token.
 *
 * @param familyId all tokens descending from one login share it
 * @param usedAt null until the token has been rotated
 * @param familyExpiresAt absolute end of the login chain, fixed at login;
 *        rotation never issues a token that expires after it
 */
public record Refresh(byte[] idHash, UUID userId, UUID familyId, Instant usedAt, Instant expiresAt,
        Instant familyExpiresAt) {

    Refresh copy() {
        return new Refresh(idHash.clone(), userId, familyId, usedAt, expiresAt, familyExpiresAt);
    }
}
