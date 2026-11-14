package io.github.brianhuanggg.authkit;

import static org.assertj.core.api.Assertions.assertThat;

import com.sun.net.httpserver.HttpServer;
import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import org.junit.jupiter.api.Test;

class HealthcheckTest {

    @Test
    void urlUsesThePortOfAddr() {
        assertThat(Healthcheck.url(":9000")).isEqualTo("http://127.0.0.1:9000/readyz");
        assertThat(Healthcheck.url("0.0.0.0:9001")).isEqualTo("http://127.0.0.1:9001/readyz");
        assertThat(Healthcheck.url(null)).isEqualTo("http://127.0.0.1:8080/readyz");
        assertThat(Healthcheck.url("")).isEqualTo("http://127.0.0.1:8080/readyz");
    }

    private static int check(int status) throws Exception {
        HttpServer server = HttpServer.create(new InetSocketAddress(InetAddress.getLoopbackAddress(), 0), 0);
        server.createContext("/readyz", ex -> {
            ex.sendResponseHeaders(status, -1);
            ex.close();
        });
        server.start();
        try {
            return Healthcheck.run("http://127.0.0.1:" + server.getAddress().getPort() + "/readyz",
                    new PrintStream(new ByteArrayOutputStream()));
        } finally {
            server.stop(0);
        }
    }

    @Test
    void exitsZeroOn200() throws Exception {
        assertThat(check(200)).isZero();
    }

    @Test
    void exitsOneOnNon200() throws Exception {
        assertThat(check(503)).isEqualTo(1);
    }

    @Test
    void exitsOneWhenNothingListens() {
        assertThat(Healthcheck.run("http://127.0.0.1:1/readyz", new PrintStream(new ByteArrayOutputStream())))
                .isEqualTo(1);
    }
}
