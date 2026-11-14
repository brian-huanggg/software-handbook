package io.github.brianhuanggg.authkit.web;

import io.github.brianhuanggg.authkit.ratelimit.ClientIp;
import jakarta.servlet.FilterChain;
import jakarta.servlet.ServletException;
import jakarta.servlet.ServletOutputStream;
import jakarta.servlet.WriteListener;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import jakarta.servlet.http.HttpServletResponseWrapper;
import java.io.IOException;
import java.security.SecureRandom;
import java.util.HexFormat;
import java.util.Set;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.MDC;
import org.springframework.core.Ordered;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;
import org.springframework.web.filter.OncePerRequestFilter;

/**
 * The outermost filter: gives every request an ID and logs one line per
 * request, so even requests rejected by a later filter (413, 429, 403) are
 * logged with their ID.
 *
 * <p>The ID is the caller's {@code X-Request-Id} when it is short, printable
 * ASCII (so it cannot inject log lines), otherwise a fresh random one. It goes
 * into the response and into the logging {@code MDC} as {@code request_id}, so
 * every log record written while handling the request carries it.
 *
 * <p>The access line has the method, path, status, size, duration and client
 * IP. Only the path is logged, never the query string, which could carry
 * secrets. Probe paths are not logged. The size counts bytes written through
 * the response stream, which is how all JSON responses here are written.
 */
@Component
@Order(Ordered.HIGHEST_PRECEDENCE)
public class RequestLogFilter extends OncePerRequestFilter {

    public static final String HEADER = "X-Request-Id";
    public static final String MDC_KEY = "request_id";

    private static final Logger log = LoggerFactory.getLogger(RequestLogFilter.class);
    private static final Set<String> PROBES = Set.of("/healthz", "/readyz");
    private static final SecureRandom RANDOM = new SecureRandom();
    private static final int MAX_ID_LENGTH = 64;

    @Override
    protected void doFilterInternal(HttpServletRequest request, HttpServletResponse response, FilterChain chain)
            throws ServletException, IOException {
        String id = request.getHeader(HEADER);
        if (!validId(id)) {
            id = newId();
        }
        response.setHeader(HEADER, id);
        MDC.put(MDC_KEY, id);

        CountingResponse counted = new CountingResponse(response);
        long start = System.nanoTime();
        boolean failed = false;
        try {
            chain.doFilter(request, counted);
        } catch (IOException | ServletException | RuntimeException e) {
            failed = true;
            throw e;
        } finally {
            // MDC still holds the ID here, so the access line carries it too.
            String path = request.getRequestURI().substring(request.getContextPath().length());
            if (!PROBES.contains(path)) {
                logRequest(request, path, failed ? 500 : response.getStatus(), counted.bytes(),
                        (System.nanoTime() - start) / 1_000_000);
            }
            MDC.remove(MDC_KEY);
        }
    }

    private static void logRequest(HttpServletRequest request, String path, int status, long bytes, long millis) {
        String ip = ClientIp.of(request);
        // The fields are both in the message (readable as text) and key-value
        // pairs (separate fields in JSON logs).
        log.atInfo()
                .addKeyValue("method", request.getMethod())
                .addKeyValue("path", path)
                .addKeyValue("status", status)
                .addKeyValue("bytes", bytes)
                .addKeyValue("duration_ms", millis)
                .addKeyValue("ip", ip)
                .log("request method={} path={} status={} bytes={} duration_ms={} ip={}",
                        request.getMethod(), path, status, bytes, millis, ip);
    }

    static boolean validId(String id) {
        if (id == null || id.isEmpty() || id.length() > MAX_ID_LENGTH) {
            return false;
        }
        for (int i = 0; i < id.length(); i++) {
            char c = id.charAt(i);
            if (c < 0x21 || c > 0x7e) {
                return false;
            }
        }
        return true;
    }

    private static String newId() {
        byte[] b = new byte[8];
        RANDOM.nextBytes(b);
        return HexFormat.of().formatHex(b);
    }

    /** Counts the bytes written to the response stream. */
    private static final class CountingResponse extends HttpServletResponseWrapper {

        private CountingStream stream;

        CountingResponse(HttpServletResponse response) {
            super(response);
        }

        long bytes() {
            return stream == null ? 0 : stream.count;
        }

        @Override
        public ServletOutputStream getOutputStream() throws IOException {
            if (stream == null) {
                stream = new CountingStream(super.getOutputStream());
            }
            return stream;
        }
    }

    private static final class CountingStream extends ServletOutputStream {

        private final ServletOutputStream out;
        long count;

        CountingStream(ServletOutputStream out) {
            this.out = out;
        }

        @Override
        public void write(int b) throws IOException {
            out.write(b);
            count++;
        }

        @Override
        public void write(byte[] b, int off, int len) throws IOException {
            out.write(b, off, len);
            count += len;
        }

        @Override
        public void flush() throws IOException {
            out.flush();
        }

        @Override
        public void close() throws IOException {
            out.close();
        }

        @Override
        public boolean isReady() {
            return out.isReady();
        }

        @Override
        public void setWriteListener(WriteListener listener) {
            out.setWriteListener(listener);
        }
    }
}
