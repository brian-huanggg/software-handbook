package io.github.brianhuanggg.authkit.password;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.nio.charset.StandardCharsets;
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

    // Made by the Go password package (password "cross-check pw ✓", pepper "cross pepper").
    private static final String GO_PLAIN =
            "$argon2id$v=19$m=19456,t=2,p=1$jJW6n91O+gb6Nn2qM88t2g$wPHu0cVpXY4xZP8y4Fo9dX7EO9al4cQ0ihkk7TCt8hE";
    private static final String GO_PEPPER =
            "$argon2id$v=19$m=19456,t=2,p=1,k=a1aaaa0d0755$/oSNz5KL5D0PvQ1dNFj+mQ$bHhngJpKzGF8vff3tytxw2quSyZnMKZU/XR1wiErLZ8";

    @Test
    void verifiesHashesMadeByGo() {
        assertThat(plain().matches("cross-check pw ✓", GO_PLAIN)).isTrue();
        assertThat(plain().matches("other", GO_PLAIN)).isFalse();
        assertThat(peppered("cross pepper").matches("cross-check pw ✓", GO_PEPPER)).isTrue();
        assertThat(plain().upgradeEncoding(GO_PLAIN)).isFalse();
        assertThat(peppered("cross pepper").upgradeEncoding(GO_PEPPER)).isFalse();
        // Same pepper id as Go derives.
        assertThat(peppered("cross pepper").encode("x")).contains(",k=a1aaaa0d0755$");
    }

    @Test
    void rejectsNonPositiveConcurrency() {
        assertThatThrownBy(() -> new PepperedArgon2Encoder(null, List.of(), 0))
                .isInstanceOf(IllegalArgumentException.class);
    }
}
