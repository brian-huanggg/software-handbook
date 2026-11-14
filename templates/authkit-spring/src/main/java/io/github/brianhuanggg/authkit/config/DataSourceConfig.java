package io.github.brianhuanggg.authkit.config;

import com.zaxxer.hikari.HikariDataSource;
import javax.sql.DataSource;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

/** Builds the pool from {@code DATABASE_URL}, the same variable the Go API reads. */
@Configuration(proxyBeanMethods = false)
class DataSourceConfig {

    @Bean(destroyMethod = "close")
    DataSource dataSource(AuthkitProperties props) {
        DatabaseUrl db = props.database();
        // Setters on a no-arg HikariDataSource defer the first connection.
        // Flyway then connects at startup, so a bad database still fails fast.
        HikariDataSource ds = new HikariDataSource();
        ds.setJdbcUrl(db.jdbcUrl());
        ds.setUsername(db.username());
        ds.setPassword(db.password());
        return ds;
    }
}
