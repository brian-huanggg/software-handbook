ARG GO_VERSION=1.26
FROM registry.internal.example/golden/go-builder:${GO_VERSION} AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/shortener .

FROM registry.internal.example/golden/go-static:nonroot
COPY --from=build /out/shortener /shortener
COPY --from=docker.io/library/busybox:latest /bin/busybox /bin/busybox
ENTRYPOINT ["/shortener"]
