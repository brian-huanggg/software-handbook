FROM registry.internal.example/golden/go-builder@sha256:1111111111111111111111111111111111111111111111111111111111111111 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/shortener .

FROM registry.internal.example/golden/go-static@sha256:2222222222222222222222222222222222222222222222222222222222222222
COPY --from=build /out/shortener /shortener
ENTRYPOINT ["/shortener"]
