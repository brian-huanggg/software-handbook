package io.github.brianhuanggg.authkit;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.context.properties.ConfigurationPropertiesScan;

@SpringBootApplication
@ConfigurationPropertiesScan
public class AuthkitApplication {

    public static void main(String[] args) {
        SpringApplication.run(AuthkitApplication.class, args);
    }
}
