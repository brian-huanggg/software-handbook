package io.github.brianhuanggg.authkit;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;

@SpringBootApplication
@ConfigurationPropertiesScan
public class AuthkitApplication {

    public static void main(String[] args) {
        if (args.length > 0 && args[0].equals("healthcheck")) {
            System.exit(Healthcheck.run(System.getenv("ADDR")));
        }
        SpringApplication.run(AuthkitApplication.class, args);
    }
}
