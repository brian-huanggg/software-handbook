# Builder: full toolchain, never shipped.
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
# CGO off: a static binary has no libc dependency, which `static` does not have either.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/shortener .

# Production: no shell, no package manager, runs as UID 65532.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/shortener /shortener
EXPOSE 8080
# Exec form is mandatory: there is no /bin/sh to interpret a shell-form command.
ENTRYPOINT ["/shortener"]
HEALTHCHECK --interval=10s --timeout=3s CMD ["/shortener", "-healthcheck"]
