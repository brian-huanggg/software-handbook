package io.github.brianhuanggg.authkit.session;

import java.time.Instant;
import java.util.UUID;

/** @param idHash SHA-256 of the raw session ID; the raw ID exists only in the client's cookie */
public record Session(byte[] idHash, UUID userId, Instant expiresAt) {
}
