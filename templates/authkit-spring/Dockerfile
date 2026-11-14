FROM eclipse-temurin:21-jdk@sha256:0a3f329821dcadf54b2a3033a15e2e9f007069a9db43d7f51897e6b80e726b07 AS build
WORKDIR /src
# Dependencies first, so source edits do not re-download them.
COPY .mvn .mvn
COPY mvnw pom.xml ./
RUN ./mvnw -B -q dependency:go-offline
COPY src src
RUN ./mvnw -B -q -DskipTests package && cp target/authkit-spring-*.jar /app.jar

# Distroless: no shell, no package manager, runs as non-root (uid 65532).
# The Docker health check runs `java -jar app.jar healthcheck` because there is no curl.
FROM gcr.io/distroless/java21-debian12:nonroot@sha256:7e37784d94dccbf5ccb195c73b295f5ad00cd266512dfbac12eb9c3c28f8077d
COPY --from=build /app.jar /app/app.jar
EXPOSE 8080
# Every password hash allocates 19 MiB outside the request path, so leave
# headroom above the heap: see "Known limitations" in the README.
ENTRYPOINT ["java", "-XX:MaxRAMPercentage=70", "-XX:+ExitOnOutOfMemoryError", "-jar", "/app/app.jar"]
