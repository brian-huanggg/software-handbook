package io.github.brianhuanggg.authkit.config;

import java.net.InetAddress;
import java.net.UnknownHostException;
import org.springframework.boot.web.server.WebServerFactoryCustomizer;
import org.springframework.boot.web.server.servlet.ConfigurableServletWebServerFactory;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/** Applies {@code ADDR} ({@code :8080} or {@code host:port}), the Go listen address format. */
@Configuration(proxyBeanMethods = false)
class ListenAddressConfig {

    @Bean
    WebServerFactoryCustomizer<ConfigurableServletWebServerFactory> listenAddress(AuthkitProperties props) {
        String addr = props.addr();
        int colon = addr.lastIndexOf(':');
        if (colon < 0) {
            throw new IllegalArgumentException("ADDR must be [host]:port (got \"" + addr + "\")");
        }
        String host = addr.substring(0, colon);
        int port;
        try {
            port = Integer.parseInt(addr.substring(colon + 1));
        } catch (NumberFormatException e) {
            throw new IllegalArgumentException("ADDR must be [host]:port (got \"" + addr + "\")");
        }
        if (port < 0 || port > 65535) {
            throw new IllegalArgumentException("ADDR port out of range (got " + port + ")");
        }
        return factory -> {
            factory.setPort(port);
            if (!host.isEmpty()) {
                try {
                    factory.setAddress(InetAddress.getByName(host));
                } catch (UnknownHostException e) {
                    throw new IllegalArgumentException("ADDR host is unknown (got \"" + host + "\")", e);
                }
            }
        };
    }
}
