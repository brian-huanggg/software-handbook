package io.github.brianhuanggg.authkit;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;

/**
 * {@code java -jar app.jar healthcheck}: asks the running API for
 * {@code /readyz} and exits 0 or 1. The container has no shell and no curl, so
 * the Docker health check runs the app itself, like the Go image does.
 */
final class Healthcheck {

    private Healthcheck() {
    }

    static String url(String addr) {
        int colon = addr == null ? -1 : addr.lastIndexOf(':');
        String port = colon >= 0 ? addr.substring(colon + 1) : "";
        return "http://127.0.0.1:" + (port.isEmpty() ? "8080" : port) + "/readyz";
    }

    /** @return the process exit code */
    static int run(String addr) {
        return run(url(addr), System.err);
    }

    static int run(String url, java.io.PrintStream err) {
        try (HttpClient client = HttpClient.newBuilder().connectTimeout(Duration.ofSeconds(2)).build()) {
            HttpResponse<Void> res = client.send(
                    HttpRequest.newBuilder(URI.create(url)).timeout(Duration.ofSeconds(2)).GET().build(),
                    HttpResponse.BodyHandlers.discarding());
            if (res.statusCode() != 200) {
                err.println("readyz returned " + res.statusCode());
                return 1;
            }
            return 0;
        } catch (IOException e) {
            err.println(e);
            return 1;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return 1;
        }
    }
}
