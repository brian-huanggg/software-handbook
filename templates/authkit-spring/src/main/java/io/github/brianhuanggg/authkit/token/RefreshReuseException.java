package io.github.brianhuanggg.authkit.token;

import java.util.UUID;

/**
 * An already-rotated token was presented again (or lost a race). The whole
 * family has been revoked. Names the family and its owner so the caller can log them.
 */
public class RefreshReuseException extends InvalidRefreshException {

    private final UUID familyId;
    private final UUID userId;

    RefreshReuseException(UUID familyId, UUID userId) {
        super("refresh token reuse detected");
        this.familyId = familyId;
        this.userId = userId;
    }

    public UUID familyId() {
        return familyId;
    }

    public UUID userId() {
        return userId;
    }
}
