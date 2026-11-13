FROM registry.internal.example/golden/go-static:1.4
COPY shortener /shortener
ENTRYPOINT ["/shortener"]
