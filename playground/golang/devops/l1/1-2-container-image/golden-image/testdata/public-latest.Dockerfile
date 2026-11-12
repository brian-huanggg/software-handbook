FROM golang:latest AS build
COPY . .
RUN go build -o /out/shortener .

FROM debian:13-slim
COPY --from=build /out/shortener /shortener
ENTRYPOINT ["/shortener"]
