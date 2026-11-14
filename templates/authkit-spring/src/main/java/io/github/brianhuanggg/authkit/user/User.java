package io.github.brianhuanggg.authkit.user;

import java.time.Instant;
import java.util.UUID;

public record User(UUID id, String email, String passwordHash, String role, Instant createdAt) {
}
