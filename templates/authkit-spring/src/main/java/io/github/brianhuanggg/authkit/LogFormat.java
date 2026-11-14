package io.github.brianhuanggg.authkit;

/**
 * Maps {@code LOG_FORMAT} to Spring Boot's structured logging, before Boot
 * configures logging. {@code json} switches the console to Boot's
 * {@code logstash} JSON format, which includes the MDC ({@code request_id})
 * and the key-value pairs of the access log as separate fields. Anything else
 * leaves the default text format. An invalid value is rejected later, with the
 * other settings, by {@code AuthkitProperties}.
 */
final class LogFormat {

    static final String PROPERTY = "logging.structured.format.console";
    static final String JSON_FORMAT = "logstash";

    private LogFormat() {
    }

    /** Sets the Boot property for {@code json}, unless the operator already set it. */
    static void apply(String logFormat) {
        if (logFormat != null && logFormat.strip().equals("json") && System.getProperty(PROPERTY) == null) {
            System.setProperty(PROPERTY, JSON_FORMAT);
        }
    }
}
