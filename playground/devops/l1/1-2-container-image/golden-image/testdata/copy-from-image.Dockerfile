FROM registry.internal.example/golden/go-static@sha256:2222222222222222222222222222222222222222222222222222222222222222
COPY shortener /shortener
COPY --from=docker.io/library/busybox:latest /bin/busybox /bin/busybox
ENTRYPOINT ["/shortener"]
