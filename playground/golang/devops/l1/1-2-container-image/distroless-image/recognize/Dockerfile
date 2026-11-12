# AI-generated. Not part of the example's tests: nothing builds this file by default.
FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/shortener .

FROM gcr.io/distroless/static-debian13
COPY --from=build /out/shortener /shortener
EXPOSE 8080
HEALTHCHECK CMD curl -f http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/shortener"]
