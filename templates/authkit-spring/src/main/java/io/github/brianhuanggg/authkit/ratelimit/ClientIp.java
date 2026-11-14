package io.github.brianhuanggg.authkit.ratelimit;

import jakarta.servlet.http.HttpServletRequest;
import java.net.Inet6Address;
import java.net.InetAddress;
import java.net.UnknownHostException;
import java.util.HexFormat;

/**
 * Keys a request by its TCP peer address. IPv6 is keyed by its /64, since one
 * subscriber usually owns a whole /64 and could otherwise rotate addresses.
 *
 * <p>Behind a reverse proxy every client shares the proxy's address: derive
 * the IP from the proxy's header there, and only trust that header when the
 * proxy is the sole way in.
 */
public final class ClientIp {

    private ClientIp() {
    }

    public static String of(HttpServletRequest request) {
        return key(request.getRemoteAddr());
    }

    static String key(String remoteAddr) {
        // Only literals reach getByName, so it never does a DNS lookup.
        if (remoteAddr == null || !(remoteAddr.indexOf(':') >= 0 || remoteAddr.matches("[0-9.]+"))) {
            return String.valueOf(remoteAddr);
        }
        try {
            InetAddress ip = InetAddress.getByName(remoteAddr);
            if (ip instanceof Inet6Address v6) {
                return HexFormat.of().formatHex(v6.getAddress(), 0, 8) + "::/64";
            }
            return ip.getHostAddress(); // IPv4, also IPv4-mapped IPv6
        } catch (UnknownHostException e) {
            return remoteAddr;
        }
    }
}
