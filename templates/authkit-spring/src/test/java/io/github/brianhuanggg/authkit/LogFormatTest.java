package io.github.brianhuanggg.authkit;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Test;

class LogFormatTest {

    @AfterEach
    void reset() {
        System.clearProperty(LogFormat.PROPERTY);
    }

    @Test
    void jsonSwitchesToStructuredLogging() {
        LogFormat.apply("json");
        assertThat(System.getProperty(LogFormat.PROPERTY)).isEqualTo("logstash");
    }

    @Test
    void textAndUnsetKeepTheDefault() {
        for (String v : new String[] {null, "", "  ", "text", "yaml"}) {
            LogFormat.apply(v);
            assertThat(System.getProperty(LogFormat.PROPERTY)).as("LOG_FORMAT=%s", v).isNull();
        }
    }

    @Test
    void anOperatorChoiceWins() {
        System.setProperty(LogFormat.PROPERTY, "ecs");
        LogFormat.apply("json");
        assertThat(System.getProperty(LogFormat.PROPERTY)).isEqualTo("ecs");
    }
}
