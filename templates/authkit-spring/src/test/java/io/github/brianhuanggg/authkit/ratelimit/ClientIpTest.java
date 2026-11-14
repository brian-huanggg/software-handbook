package io.github.brianhuanggg.authkit.ratelimit;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;

class ClientIpTest {

    @Test
    void ipv4IsKeyedWhole() {
        assertThat(ClientIp.key("203.0.113.7")).isEqualTo("203.0.113.7");
        assertThat(ClientIp.key("203.0.113.8")).isNotEqualTo(ClientIp.key("203.0.113.7"));
    }

    @Test
    void ipv6IsKeyedBySlash64() {
        String a = ClientIp.key("2001:db8:1:2:aaaa:bbbb:cccc:dddd");
        String b = ClientIp.key("2001:db8:1:2:1111:2222:3333:4444");
        assertThat(a).isEqualTo(b).isEqualTo("20010db800010002::/64");
        assertThat(ClientIp.key("2001:db8:1:3::1")).as("another /64").isNotEqualTo(a);
    }

    @Test
    void mappedIpv4MatchesPlainIpv4() {
        assertThat(ClientIp.key("::ffff:203.0.113.7")).isEqualTo(ClientIp.key("203.0.113.7"));
    }

    @Test
    void nonLiteralIsKeyedAsIs() {
        assertThat(ClientIp.key("not-an-ip")).isEqualTo("not-an-ip");
    }
}
