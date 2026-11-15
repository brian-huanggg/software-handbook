package io.github.brianhuanggg.authkit.password;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.List;
import org.junit.jupiter.api.Test;

class PepperedArgon2EncoderTest {

    private static byte[] b(String s) {
        return s.getBytes(StandardCharsets.UTF_8);
    }

    private static PepperedArgon2Encoder plain() {
        return new PepperedArgon2Encoder(null, List.of(), 2);
    }

    private static PepperedArgon2Encoder peppered(String pepper, String... previous) {
        return new PepperedArgon2Encoder(b(pepper),
                java.util.Arrays.stream(previous).map(PepperedArgon2EncoderTest::b).toList(), 2);
    }

    @Test
    void roundTrip() {
        var e = plain();
        String enc = e.encode("correct horse");
        assertThat(enc).startsWith("$argon2id$v=19$m=19456,t=2,p=1$");
        assertThat(e.matches("correct horse", enc)).isTrue();
    }

    @Test
    void wrongPassword() {
        var e = plain();
        assertThat(e.matches("wrong", e.encode("right"))).isFalse();
    }

    @Test
    void saltIsRandom() {
        var e = plain();
        assertThat(e.encode("same")).isNotEqualTo(e.encode("same"));
    }

    @Test
    void invalidEncodedIsFalse() {
        var e = plain();
        for (String enc : List.of(
                "",
                "plaintext",
                "$bcrypt$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA",
                "$argon2id$v=1$m=19456,t=2,p=1$c2FsdA$aGFzaA",
                "$argon2id$v=19$bogus$c2FsdA$aGFzaA",
                "$argon2id$v=19$m=19456,t=2,p=1$!!!$aGFzaA",
                "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$!!!",
                "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA==$aGFzaA",
                "$argon2id$v=19$m=19456,t=2,p=1,k=$c2FsdA$aGFzaA")) {
            assertThat(e.matches("x", enc)).as(enc).isFalse();
            assertThat(e.upgradeEncoding(enc)).as(enc).isTrue();
        }
    }

    @Test
    void rejectsExcessiveParams() {
        var e = plain();
        for (String enc : List.of(
                "$argon2id$v=19$m=4194304,t=2,p=1$c2FsdA$aGFzaA", // 4 GiB
                "$argon2id$v=19$m=19456,t=1000,p=1$c2FsdA$aGFzaA",
                "$argon2id$v=19$m=19456,t=2,p=255$c2FsdA$aGFzaA",
                "$argon2id$v=19$m=99999999999,t=2,p=1$c2FsdA$aGFzaA")) {
            assertThat(e.matches("x", enc)).as(enc).isFalse();
        }
    }

    @Test
    void rejectsTrailingParams() {
        var e = plain();
        String bad = e.encode("pw").replace("p=1", "p=1,x=9");
        assertThat(e.matches("pw", bad)).isFalse();
    }

    @Test
    void pepper() {
        var withPepper = peppered("pepper");
        String enc = withPepper.encode("pw");
        assertThat(withPepper.matches("pw", enc)).isTrue();
        assertThatThrownBy(() -> plain().matches("pw", enc)).isInstanceOf(UnknownPepperException.class);
        assertThatThrownBy(() -> peppered("other").matches("pw", enc))
                .isInstanceOf(UnknownPepperException.class);
    }

    @Test
    void pepperRotation() {
        String enc = peppered("old pepper").encode("pw");

        var rotated = peppered("new pepper", "old pepper");
        assertThat(rotated.matches("pw", enc)).isTrue();
        assertThat(rotated.upgradeEncoding(enc)).isTrue();
        String fresh = rotated.encode("pw");
        assertThat(rotated.upgradeEncoding(fresh)).isFalse();
        assertThat(fresh).doesNotContain("pepper");

        assertThatThrownBy(() -> peppered("new pepper").matches("pw", enc))
                .isInstanceOf(UnknownPepperException.class);
    }

    @Test
    void pepperAddedLater() {
        String enc = plain().encode("pw");
        assertThat(enc).doesNotContain(",k=");
        var e = peppered("pepper");
        assertThat(e.matches("pw", enc)).isTrue();
        assertThat(e.upgradeEncoding(enc)).isTrue();
    }

    @Test
    void upgradeEncodingOldParams() {
        var e = plain();
        String enc = e.encode("pw");
        assertThat(e.upgradeEncoding(enc)).isFalse();
        assertThat(e.upgradeEncoding(enc.replace("t=2", "t=1"))).isTrue();
        assertThat(e.upgradeEncoding("garbage")).isTrue();
    }

    // Made by an independent implementation of the same format (password "cross-check pw ✓", pepper "cross pepper").
    private static final String FOREIGN_PLAIN =
            "$argon2id$v=19$m=19456,t=2,p=1$jJW6n91O+gb6Nn2qM88t2g$wPHu0cVpXY4xZP8y4Fo9dX7EO9al4cQ0ihkk7TCt8hE";
    private static final String FOREIGN_PEPPER =
            "$argon2id$v=19$m=19456,t=2,p=1,k=a1aaaa0d0755$/oSNz5KL5D0PvQ1dNFj+mQ$bHhngJpKzGF8vff3tytxw2quSyZnMKZU/XR1wiErLZ8";

    @Test
    void verifiesHashesFromOtherImplementation() {
        assertThat(plain().matches("cross-check pw ✓", FOREIGN_PLAIN)).isTrue();
        assertThat(plain().matches("other", FOREIGN_PLAIN)).isFalse();
        assertThat(peppered("cross pepper").matches("cross-check pw ✓", FOREIGN_PEPPER)).isTrue();
        assertThat(plain().upgradeEncoding(FOREIGN_PLAIN)).isFalse();
        assertThat(peppered("cross pepper").upgradeEncoding(FOREIGN_PEPPER)).isFalse();
        // Pepper id is derived the same way in the other implementation.
        assertThat(peppered("cross pepper").encode("x")).contains(",k=a1aaaa0d0755$");
    }

    @Test
    void rejectsNonPositiveConcurrency() {
        assertThatThrownBy(() -> new PepperedArgon2Encoder(null, List.of(), 0))
                .isInstanceOf(IllegalArgumentException.class);
    }

    // ---- admission control ----

    @Test
    void slotsAreWeightedByMemory() {
        int m = PepperedArgon2Encoder.MEMORY_KIB;
        assertThat(PepperedArgon2Encoder.slotsFor(1, 8)).isEqualTo(1);
        assertThat(PepperedArgon2Encoder.slotsFor(m, 8)).isEqualTo(1);
        assertThat(PepperedArgon2Encoder.slotsFor(m + 1, 8)).isEqualTo(2);
        // 256 MiB / 19 MiB, rounded up, is 14 slots ...
        assertThat(PepperedArgon2Encoder.slotsFor(PepperedArgon2Encoder.MAX_MEMORY_KIB, 32)).isEqualTo(14);
        // ... but never more than exist, or the hash could never run.
        assertThat(PepperedArgon2Encoder.slotsFor(PepperedArgon2Encoder.MAX_MEMORY_KIB, 8)).isEqualTo(8);
    }

    @Test
    void busyWhenNoSlotFreesWithinTheWait() {
        var e = new PepperedArgon2Encoder(null, List.of(), 1, Duration.ofMillis(50));
        String hash = e.encode("pw");
        e.slots.acquireUninterruptibly();
        try {
            assertThatThrownBy(() -> e.encode("pw")).isInstanceOf(PasswordHashingBusyException.class);
            assertThatThrownBy(() -> e.matches("pw", hash)).isInstanceOf(PasswordHashingBusyException.class);
        } finally {
            e.slots.release();
        }
        assertThat(e.matches("pw", hash)).as("works again once a slot is free").isTrue();
        assertThat(e.slots.availablePermits()).as("every slot given back").isEqualTo(1);
    }

    @Test
    void heavierStoredHashNeedsMoreSlots() {
        var e = new PepperedArgon2Encoder(null, List.of(), 2, Duration.ofMillis(50));
        String normal = e.encode("pw");
        // Same string with twice the memory cost: it needs both slots.
        String heavy = normal.replace("m=" + PepperedArgon2Encoder.MEMORY_KIB,
                "m=" + 2 * PepperedArgon2Encoder.MEMORY_KIB);
        e.slots.acquireUninterruptibly();
        try {
            assertThat(e.matches("pw", normal)).as("one slot is enough").isTrue();
            assertThatThrownBy(() -> e.matches("pw", heavy)).isInstanceOf(PasswordHashingBusyException.class);
        } finally {
            e.slots.release();
        }
        assertThat(e.slots.availablePermits()).isEqualTo(2);
    }
}
