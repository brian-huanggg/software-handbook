package io.github.brianhuanggg.authkit.password;

import java.nio.charset.StandardCharsets;
import java.security.GeneralSecurityException;
import java.security.MessageDigest;
import java.security.SecureRandom;
import java.util.Arrays;
import java.util.Base64;
import java.util.HashMap;
import java.util.HexFormat;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.concurrent.Semaphore;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;
import org.bouncycastle.crypto.generators.Argon2BytesGenerator;
import org.bouncycastle.crypto.params.Argon2Parameters;
import org.springframework.security.crypto.password.PasswordEncoder;

/**
 * Argon2id with an optional HMAC pepper, writing the same PHC strings as the Go
 * {@code password} package, so either API can verify the other's hashes:
 *
 * <pre>$argon2id$v=19$m=...,t=...,p=...[,k=&lt;pepper id&gt;]$salt$hash</pre>
 *
 * Not wrapped in {@code DelegatingPasswordEncoder}: no {@code {id}} prefix.
 * BouncyCastle is used directly because Spring's {@code Argon2PasswordEncoder}
 * cannot carry the {@code k} parameter or cap parameters read from a stored hash.
 *
 * <p>{@link #matches} returns false for a malformed hash and throws
 * {@link UnknownPepperException} when the hash names a pepper that is not
 * configured: a configuration error, not a wrong password.
 */
public final class PepperedArgon2Encoder implements PasswordEncoder {

    // OWASP Password Storage Cheat Sheet minimum. Written into every hash,
    // so raising them later still verifies old hashes.
    static final int MEMORY_KIB = 19 * 1024;
    static final int TIME_COST = 2;
    static final int PARALLELISM = 1;
    static final int SALT_LEN = 16;
    static final int KEY_LEN = 32;

    // Upper bounds on parameters read from a stored hash, so a corrupt or
    // tampered row cannot make matches() allocate gigabytes or spin for minutes.
    static final int MAX_MEMORY_KIB = 256 * 1024;
    static final int MAX_TIME_COST = 10;
    static final int MAX_PARALLELISM = 16;
    // BouncyCastle rejects tags shorter than 4 bytes.
    private static final int MIN_KEY_LEN = 4;

    private static final int ARGON2_VERSION = 19;
    private static final Pattern PARAMS = Pattern.compile(
            "m=([1-9]\\d{0,8}),t=([1-9]\\d{0,8}),p=([1-9]\\d{0,8})");

    private static final SecureRandom RANDOM = new SecureRandom();
    private static final Base64.Encoder B64 = Base64.getEncoder().withoutPadding();

    private final String pepperId; // null = no pepper
    private final Map<String, byte[]> peppers; // id -> pepper, current one included
    private final Semaphore slots;

    /**
     * @param pepper   signs new hashes; null or empty means none. Keep it outside the DB.
     * @param previous only verify, so a pepper can be rotated: make the new one
     *                 current, keep the old one here until {@link #upgradeEncoding}
     *                 has moved every hash off it, then drop it.
     * @param maxConcurrent at most this many hash or verify calls run at once.
     *                 Each call allocates 19 MiB, so without a cap a burst of
     *                 logins can exhaust memory. Pick about the CPU count.
     */
    public PepperedArgon2Encoder(byte[] pepper, List<byte[]> previous, int maxConcurrent) {
        if (maxConcurrent <= 0) {
            throw new IllegalArgumentException("maxConcurrent must be > 0");
        }
        this.slots = new Semaphore(maxConcurrent, true);
        Map<String, byte[]> all = new HashMap<>();
        String current = null;
        if (pepper != null && pepper.length > 0) {
            current = pepperId(pepper);
            all.put(current, pepper.clone());
        }
        for (byte[] p : previous) {
            if (p != null && p.length > 0) {
                all.putIfAbsent(pepperId(p), p.clone());
            }
        }
        this.peppers = Map.copyOf(all);
        this.pepperId = current;
    }

    /** Names a pepper by a truncated hash of it, never the pepper itself. */
    static String pepperId(byte[] pepper) {
        try {
            MessageDigest sha = MessageDigest.getInstance("SHA-256");
            sha.update("authkit-pepper:".getBytes(StandardCharsets.UTF_8));
            sha.update(pepper);
            return HexFormat.of().formatHex(sha.digest(), 0, 6);
        } catch (GeneralSecurityException e) {
            throw new IllegalStateException(e);
        }
    }

    /** Binds the password to the pepper with HMAC-SHA256. */
    private static byte[] applyPepper(byte[] pepper, CharSequence password) {
        byte[] pw = password.toString().getBytes(StandardCharsets.UTF_8);
        if (pepper == null) {
            return pw;
        }
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(pepper, "HmacSHA256"));
            return mac.doFinal(pw);
        } catch (GeneralSecurityException e) {
            throw new IllegalStateException(e);
        }
    }

    private byte[] argon2id(byte[] secret, byte[] salt, int t, int m, int p, int keyLen) {
        slots.acquireUninterruptibly();
        try {
            Argon2Parameters params = new Argon2Parameters.Builder(Argon2Parameters.ARGON2_id)
                    .withVersion(Argon2Parameters.ARGON2_VERSION_13)
                    .withIterations(t)
                    .withMemoryAsKB(m)
                    .withParallelism(p)
                    .withSalt(salt)
                    .build();
            Argon2BytesGenerator gen = new Argon2BytesGenerator();
            gen.init(params);
            byte[] out = new byte[keyLen];
            gen.generateBytes(secret, out);
            return out;
        } finally {
            slots.release();
        }
    }

    @Override
    public String encode(CharSequence rawPassword) {
        byte[] salt = new byte[SALT_LEN];
        RANDOM.nextBytes(salt);
        byte[] pepper = pepperId == null ? null : peppers.get(pepperId);
        byte[] key = argon2id(applyPepper(pepper, rawPassword), salt,
                TIME_COST, MEMORY_KIB, PARALLELISM, KEY_LEN);
        String params = "m=" + MEMORY_KIB + ",t=" + TIME_COST + ",p=" + PARALLELISM;
        if (pepperId != null) {
            params += ",k=" + pepperId;
        }
        return "$argon2id$v=" + ARGON2_VERSION + "$" + params
                + "$" + B64.encodeToString(salt) + "$" + B64.encodeToString(key);
    }

    @Override
    public boolean matches(CharSequence rawPassword, String encoded) {
        Decoded d = Decoded.parse(encoded);
        if (d == null) {
            return false;
        }
        byte[] pepper = null;
        if (d.pepperId != null) {
            pepper = peppers.get(d.pepperId);
            if (pepper == null) {
                throw new UnknownPepperException();
            }
        }
        byte[] got = argon2id(applyPepper(pepper, rawPassword), d.salt, d.t, d.m, d.p, d.key.length);
        return MessageDigest.isEqual(got, d.key);
    }

    /**
     * True when {@code encoded} was made with other parameters or another
     * pepper than {@link #encode} uses now. After a successful match, store a
     * fresh encode of the password so old hashes upgrade as users log in.
     */
    @Override
    public boolean upgradeEncoding(String encoded) {
        Decoded d = Decoded.parse(encoded);
        return d == null || d.m != MEMORY_KIB || d.t != TIME_COST || d.p != PARALLELISM
                || d.salt.length != SALT_LEN || d.key.length != KEY_LEN
                || !Objects.equals(d.pepperId, pepperId);
    }

    /** A parsed PHC string. */
    private record Decoded(int m, int t, int p, String pepperId, byte[] salt, byte[] key) {

        /** Returns null for anything malformed or over the parameter caps. */
        static Decoded parse(String encoded) {
            if (encoded == null) {
                return null;
            }
            String[] parts = encoded.split("\\$", -1);
            if (parts.length != 6 || !parts[0].isEmpty() || !parts[1].equals("argon2id")
                    || !parts[2].equals("v=" + ARGON2_VERSION)) {
                return null;
            }
            String params = parts[3];
            String kid = null;
            int k = params.indexOf(",k=");
            if (k >= 0) {
                kid = params.substring(k + 3);
                params = params.substring(0, k);
                if (kid.isEmpty()) {
                    return null;
                }
            }
            Matcher m = PARAMS.matcher(params);
            if (!m.matches()) {
                return null;
            }
            long mem = Long.parseLong(m.group(1));
            long time = Long.parseLong(m.group(2));
            long par = Long.parseLong(m.group(3));
            if (mem > MAX_MEMORY_KIB || time > MAX_TIME_COST || par > MAX_PARALLELISM) {
                return null;
            }
            byte[] salt = rawBase64(parts[4]);
            byte[] key = rawBase64(parts[5]);
            if (salt == null || key == null || key.length < MIN_KEY_LEN) {
                return null;
            }
            return new Decoded((int) mem, (int) time, (int) par, kid, salt, key);
        }

        /** Standard alphabet, no padding, like Go's RawStdEncoding. */
        private static byte[] rawBase64(String s) {
            if (s.indexOf('=') >= 0) {
                return null;
            }
            try {
                return Base64.getDecoder().decode(s);
            } catch (IllegalArgumentException e) {
                return null;
            }
        }

        @Override
        public boolean equals(Object o) {
            return o instanceof Decoded d && m == d.m && t == d.t && p == d.p
                    && Objects.equals(pepperId, d.pepperId)
                    && Arrays.equals(salt, d.salt) && Arrays.equals(key, d.key);
        }

        @Override
        public int hashCode() {
            return Objects.hash(m, t, p, pepperId) * 31 + Arrays.hashCode(key);
        }
    }
}
