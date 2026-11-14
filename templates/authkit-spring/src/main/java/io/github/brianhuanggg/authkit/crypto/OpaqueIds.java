package io.github.brianhuanggg.authkit.crypto;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.util.Base64;

/**
 * Random bearer values that the server stores only as a hash: session IDs and
 * refresh tokens. The raw value exists only in the client's cookie; the store
 * sees its SHA-256, so a leaked database cannot be replayed. Same format as
 * the Go API: 32 random bytes, base64url without padding.
 */
public final class OpaqueIds {

    private static final int BYTES = 32;
    private static final SecureRandom RANDOM = new SecureRandom();

    private OpaqueIds() {
    }

    /** A new raw value for the client. */
    public static String generate() {
        byte[] random = new byte[BYTES];
        RANDOM.nextBytes(random);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(random);
    }

    /** The SHA-256 of a raw value: what the store keys on. */
    public static byte[] hash(String raw) {
        try {
            return MessageDigest.getInstance("SHA-256").digest(raw.getBytes(StandardCharsets.UTF_8));
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException(e);
        }
    }
}
