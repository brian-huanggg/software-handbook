# authkit-spring

The [Go authkit](../authkit) template, rebuilt on Spring Boot 4 (Spring
Security 7, Jackson 3) and Java 21. It is the same HTTP API and the same
database schema, so you can compare the two stacks, run both against one
database, and use whichever fits your team.

| | Session mode | Token mode |
|---|---|---|
| Credential | Opaque session ID in an HttpOnly cookie | Short-lived access JWT in the `Authorization` header |
| Server state | One row per session | One row per refresh token; access JWTs are stateless |
| Revocation | Immediate (delete the row) | Access JWT lives until it expires (15 min); refresh chain is revoked immediately |
| Endpoints | `/session/*` | `/token/*` |

The contract is [../authkit/api/openapi.yaml](../authkit/api/openapi.yaml). The
Go end-to-end suite in [../authkit/integration](../authkit/integration) runs
unchanged against this API and is the acceptance test (see
[Tests](#tests)).

> This is a learning and starting-point template, not a hosted product. Read
> [Known limitations](#known-limitations) before using it in production.
> [Differences from Go](#differences-from-go) explains what Spring Security
> does for you here and what stayed custom, and why.

## Quick start

Requirements: Java 21 and Docker (the tests start Postgres with
Testcontainers). Go 1.26 is only needed for `make integration`, because the
end-to-end suite is written in Go.

### Full stack in containers

```sh
make stack-up        # db, migrations, api-java (:8082). Uses ../authkit/docker-compose.yml
make stack-down
```

The Java API is an opt-in Compose service (`--profile java`) next to the Go
one. It listens on host port **8082** so both can run at once. `JWT_SECRET` is
generated for each run; export your own to keep it stable.

To use the React demo from [../authkit/web](../authkit/web) with the Java API,
point it at port 8082. Run this in `../authkit`:

```sh
WEB_API_URL=http://localhost:8082 docker compose --profile java up -d --build api-java web
open http://localhost:3000
```

### API on the host

```sh
docker compose -f ../authkit/docker-compose.yml up -d db   # Postgres only
DATABASE_URL='postgres://authkit:authkit@localhost:5432/authkit?sslmode=disable' \
JWT_SECRET=$(openssl rand -hex 32) \
CORS_ORIGINS=http://localhost:3000 \
./mvnw spring-boot:run
```

No separate migration step: Flyway applies the schema when the app starts.

### Tests

```sh
make test            # = ./mvnw test: unit tests plus MockMvc and Postgres tests (Testcontainers, postgres:16)
make integration     # starts the stack, then runs the Go end-to-end suite against :8082
```

`make integration` raises both rate-limit bursts to 1000, because the suite
logs in dozens of times from one IP. The rate limiter has its own tests.

## Configuration

All settings come from the same environment variables as the Go API, so a
Compose file or deployment can swap one for the other. They are bound to the
`AuthkitProperties` record through [application.yml](src/main/resources/application.yml).
The app refuses to start if a required one is missing or invalid, so it never
runs with an unsafe default.

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DATABASE_URL` | yes | – | Postgres URL, `postgres://user:pass@host:5432/db?sslmode=disable`. A `jdbc:postgresql:` URL also works |
| `JWT_SECRET` | yes | – | HMAC key that signs access tokens, at least 32 bytes |
| `JWT_PREVIOUS_SECRETS` | no | none | Comma-separated old keys that still verify but never sign, for rotation. Each at least 32 bytes, none a duplicate |
| `JWT_ISSUER` | no | `authkit` | `iss` claim, checked on verify |
| `PASSWORD_PEPPER` | no | none | Secret mixed into new password hashes; keep it outside the database |
| `PASSWORD_PREVIOUS_PEPPERS` | no | none | Comma-separated old peppers that still verify, for rotation |
| `CORS_ORIGINS` | no | none | Comma-separated browser origins allowed to call the API. Each must be `scheme://host[:port]`, with no path and no `*` |
| `LOG_FORMAT` | no | `text` | `json` for structured logs (one JSON object per line, Spring Boot's `logstash` format). Every request is logged (method, path, status, bytes, duration, client IP) with a `request_id`, also returned in `X-Request-Id`; probes are not logged. The compose stack uses `json` |
| `COOKIE_INSECURE` | no | off | `1` or `true` drops the `Secure` cookie flag. Development only |
| `ADDR` | no | `:8080` | Listen address, `[host]:port` |
| `RATE_LIMIT_IP_BURST` | no | `20` | Burst of the per-IP limiter on `/register` and both logins. Positive integer; it cannot be turned off |
| `RATE_LIMIT_EMAIL_BURST` | no | `5` | Burst of the per-email limiter on both logins. Positive integer |

Lifetimes are constants in the code: session 24 h
([SessionService](src/main/java/io/github/brianhuanggg/authkit/session/SessionService.java)),
refresh token 7 days idle and 30 days per login chain
([RefreshTokenService](src/main/java/io/github/brianhuanggg/authkit/token/RefreshTokenService.java)),
access token 15 min
([JwtService](src/main/java/io/github/brianhuanggg/authkit/token/JwtService.java)),
janitor every 10 min
([Janitor](src/main/java/io/github/brianhuanggg/authkit/ratelimit/Janitor.java)).

Pepper and JWT key rotation work exactly as in the Go README: set the new
value, move the old one to the `*_PREVIOUS_*` variable, and remove it once
nothing depends on it.

## HTTP API

Identical to the Go API; see [../authkit/api/openapi.yaml](../authkit/api/openapi.yaml)
and the request examples in [../authkit/README.md](../authkit/README.md#http-api)
(use port 8082).

| Method and path | Auth | Description |
|---|---|---|
| `GET /healthz` | none | Liveness check (never touches the DB) |
| `GET /readyz` | none | Readiness check: `200` when the DB answers a ping, else `503` |
| `POST /register` | none | Create an account (`email`, `password` of 8–128 bytes). Always `202`, also for an existing email |
| `POST /session/login` | none | Verify credentials, set the `sid` cookie |
| `POST /session/logout` | cookie | Delete the session, clear the cookie |
| `GET /session/me` | cookie | Return `user_id` and `roles` |
| `POST /token/login` | none | Return an access JWT, set the `refresh_token` cookie |
| `POST /token/refresh` | refresh cookie | Rotate the refresh token, return a new access JWT |
| `POST /token/logout` | refresh cookie | Revoke the whole refresh chain |
| `GET /token/me` | bearer | Return `user_id` and `roles` |

Every `POST` must carry `X-Requested-With: XMLHttpRequest`. Every error is JSON
`{"error": "..."}` and every response carries `Cache-Control: no-store`.

## Design

The behavior is the same as in Go, and the reasoning is in the
[Go README](../authkit/README.md#design). This section lists where each part
lives in Java.

### Layering

```
request
  -> RequestLogFilter, BodySizeLimitFilter, NoStoreFilter   servlet filters, ahead of Spring Security
  -> SecurityFilterChain (one of three)
       CorsFilter -> CsrfHeaderFilter -> RateLimitFilter -> authentication -> authorization
  -> controller
```

[SecurityConfig](src/main/java/io/github/brianhuanggg/authkit/config/SecurityConfig.java)
defines three chains, each with the same base: `/session/**`, `/token/**`, and
everything else (public). The order matters and is the part most worth
reading:

1. **Request log** (ID and access line), so every response is logged.
2. **Body size limit** (64 KiB, `413`), so no later filter buffers
   unbounded input. It keeps the buffered body so the rate limiter can read
   the login email and the controller can still read it.
3. **CORS** before CSRF, so a preflight is answered before the CSRF check.
4. **CSRF** (`X-Requested-With` and the `Origin` allowlist).
5. **Rate limit**, before any password hash runs.
6. **Authentication**: the `sid` cookie filter on `/session/**`, the bearer
   token filter on `/token/**`.

A missing or invalid credential gives `401`, a valid principal without the
role gives `403`, and an unexpected failure gives `500` with a generic body.
All three come from JSON handlers, so the error shape never changes.

### Sessions

Same design as Go: 32 random bytes, stored as SHA-256, 24 h expiry checked on
read, login deletes any session the request already carried (fixation), roles
read from `users` on every request. The cookie is `sid`, `HttpOnly`, `Secure`,
`SameSite=Lax`, `Path=/`.
[SessionAuthenticationFilter](src/main/java/io/github/brianhuanggg/authkit/session/SessionAuthenticationFilter.java)
puts an `AuthenticatedPrincipal` in the security context. Spring never creates
an `HttpSession`.

### Tokens

- **Access token**: HS256 JWT with `sub`, `iss`, `iat`, `exp` and `roles`, with
  a `kid` derived from the key the same way as in Go. Tokens made by one API
  verify in the other when both use the same secrets.
  [JwtService](src/main/java/io/github/brianhuanggg/authkit/token/JwtService.java)
  signs with Spring's `NimbusJwtEncoder` and verifies with a `NimbusJwtDecoder`
  that pins HS256, selects the key by `kid`, and adds checks Spring does not
  make by default. A token is rejected when it has no `kid`, no `exp`, a
  non-UUID `sub`, an `iat` in the future, a `roles` claim that is not a list
  of strings, or when `exp` has been reached. There is no clock skew.
  Every failure is the same `401` with `WWW-Authenticate: Bearer realm="authkit"`.
- **Refresh token**: 32 random bytes stored as SHA-256, in the
  `refresh_token` cookie (`HttpOnly`, `Secure`, `SameSite=Strict`,
  `Path=/token`).
- **Rotation** is one transaction in
  [PostgresRefreshStore](src/main/java/io/github/brianhuanggg/authkit/token/PostgresRefreshStore.java):
  `UPDATE ... SET used_at = ? WHERE id_hash = ? AND used_at IS NULL`, then the
  `INSERT` of the new token. Concurrent callers queue on the row lock, so
  exactly one wins. If the insert fails the update rolls back and the old token
  stays valid.
- **Reuse detection**: an already-used token, or losing the race, revokes the
  whole family. Rotation never extends past `family_expires_at`.
- Roles are read from `users` on every refresh, so a role change applies at
  the next refresh.

### Passwords

[PepperedArgon2Encoder](src/main/java/io/github/brianhuanggg/authkit/password/PepperedArgon2Encoder.java)
is a Spring `PasswordEncoder`:

- Argon2id with 19 MiB, 2 iterations, 1 thread, as a PHC string. Hashes made
  by the Go API verify here and the other way round (a test checks Go-made
  hashes, including the pepper id).
- HMAC-SHA256 pepper with a `k=<id>` parameter, so several peppers can verify
  at once. `upgradeEncoding` reports old parameters or a previous pepper, and
  the hash is replaced after a successful login. The update is conditional
  (`WHERE password_hash = :old`), so it cannot undo a password change.
- Parameters read from a stored hash are capped, so a tampered row cannot
  force a huge allocation.
- At most `availableProcessors()` hashes run at once (a fair `Semaphore`).
- There is no `DelegatingPasswordEncoder` and so no `{argon2}` prefix: the
  stored format must stay the same as Go's.

`/register` answers `202` with the same body, after the same hashing work, for
a new and an existing email. Login goes through Spring's
`DaoAuthenticationProvider`, which verifies against a dummy hash for an unknown
email, so timing and the message do not reveal which accounts exist.
Passwords are limited to 8–128 bytes. Unknown JSON fields and trailing tokens
are rejected with `400`.

### Request logging

[RequestLogFilter](src/main/java/io/github/brianhuanggg/authkit/web/RequestLogFilter.java)
is the outermost filter, so requests refused by a later filter (`403`, `413`,
`429`) are logged too. It gives each request an ID: the caller's
`X-Request-Id` if it is at most 64 printable ASCII characters (so it cannot
inject log lines), otherwise a random one. The ID is returned in the response
and put in the logging MDC, so every log line written while handling the
request carries it: as `[id]` after the level in text logs, as a `request_id`
field in JSON logs. Only the path is logged, never the query string.

`LOG_FORMAT=json` is applied in `main` before Spring Boot configures logging
([LogFormat](src/main/java/io/github/brianhuanggg/authkit/LogFormat.java)). If
you set `logging.structured.format.console` yourself, that value wins.

### Rate limiting

[RateLimitFilter](src/main/java/io/github/brianhuanggg/authkit/ratelimit/RateLimitFilter.java)
with in-memory [Bucket4j](https://github.com/bucket4j/bucket4j) buckets:

- Per client IP on `/register` and both logins: burst 20, then one request
  every 3 s. IPv6 is keyed by its /64.
- Per email (trimmed, lower-cased) on both logins: burst 5, then one attempt
  every 30 s, whether or not the account exists.
- Over the limit: `429` with `Retry-After`, before any password hash runs. A
  body the controller would reject with `400` does not spend the email limit.
- Idle buckets that have fully refilled are dropped, so memory follows the
  number of recently active keys.

### CSRF and CORS

Spring's CSRF filter is off. Instead,
[CsrfHeaderFilter](src/main/java/io/github/brianhuanggg/authkit/web/CsrfHeaderFilter.java)
applies the Go rules to every unsafe method: `X-Requested-With` is required,
and any `Origin` must be in `CORS_ORIGINS`. CORS is an explicit allowlist with
credentials, never a wildcard. `GET` handlers must never change state.

### Database

Flyway applies [V1 and V2](src/main/resources/db/migration), copies of the Go
migrations, using `JdbcClient` and no JPA. The tables are the same:
`users`, `sessions`, `refresh_tokens`. `baseline-on-migrate` at version 2 lets
Flyway adopt a schema that golang-migrate already created, which is what the
Compose stack does.

### Deployment image

[Dockerfile](Dockerfile) builds with Maven and runs on
`distroless/java21-debian12:nonroot` (uid 65532). The image has no shell, so
the Compose health check runs `java -jar /app/app.jar healthcheck`, which
calls `/readyz` and exits 0 or 1. Shutdown is graceful (10 s).

## Differences from Go

### What Spring Security and Spring Boot do

| Concern | Spring piece | Go equivalent |
|---|---|---|
| Wiring the filters in order | Three `SecurityFilterChain` beans, lambda DSL only | `net/http` middleware stacked by hand |
| Bearer token parsing and `401` flow | `oauth2ResourceServer(o -> o.jwt(...))`, Nimbus JOSE | `token.Bearer`, `golang-jwt` |
| JWT signing and key lookup | `NimbusJwtEncoder`, `NimbusJwtDecoder` | `golang-jwt` |
| Email and password check, timing protection, hash upgrade | `DaoAuthenticationProvider` with `UserDetailsPasswordService` | Written by hand in the login handler |
| CORS | `CorsFilter` with a `CorsConfigurationSource` | `middleware.CORS` |
| Turning off form login, HTTP Basic, logout, request cache, `HttpSession` | Security DSL and `SessionCreationPolicy.STATELESS` | Not applicable |
| Config binding and fail-fast validation | `@ConfigurationProperties` record | `loadConfig` |
| Migrations | Flyway | golang-migrate |
| SQL | `JdbcClient` | `pgx` |
| Janitor | `@Scheduled` | A goroutine ticker |
| Graceful shutdown | `server.shutdown=graceful` | `http.Server.Shutdown` |

### What stayed custom, and why

- **Sessions are not Spring Session.** Keeping hashed IDs in the same table as
  Go, plus the per-request role lookup, is simpler than adapting Spring
  Session's model. Spring is told to create no `HttpSession`.
- **The password encoder is custom (BouncyCastle `Argon2BytesGenerator`).**
  Spring's `Argon2PasswordEncoder` cannot add the pepper `k=` parameter, read
  previous peppers, or cap parameters from a stored hash.
- **CSRF is a custom filter**, not Spring's. Spring's CSRF protects
  form-style requests with a token; this API relies on a custom header plus an
  `Origin` allowlist, as the Go API does.
- **The body size limit and the rate limiter are custom filters.** Spring has
  no 64 KiB limit that runs before everything else, and nothing keyed by IP
  and login email. Bucket4j does the bucket math; the keys, the IPv6 /64 rule
  and the sweep are ours.
- **Refresh tokens are custom.** Spring Security has no rotating, reuse-
  detecting refresh token, and the atomic rotation is the one place where
  correctness depends on the exact SQL.
- **JWT validation has extra rules** (see [Tokens](#tokens)) so both APIs
  accept and reject the same tokens. Spring's own timestamp check accepts a
  token at the exact `exp` second and the Go parser does not.
- **The `Authentication` is our `AuthenticatedPrincipal`** in both modes, so
  controllers and `/session/me` and `/token/me` share one type.

### Behavior that differs from the Go API

- **Log field names differ.** JSON logs use Spring Boot's `logstash` layout
  (`@timestamp`, `message`, `level`, `logger_name`, ...), so `duration_ms`
  replaces Go's `duration` and the request fields are also repeated in the
  message. A log pipeline needs a separate mapping for each stack.
- **The access log counts bytes written through the response stream**, which is
  how every response here is written.
- **Startup is slower** and memory use is higher: a JVM, Flyway at boot, and
  the image is about 339 MB. After the integration suite the container used
  about 320 MiB.
- **Schema management.** Flyway owns the schema in this project and
  golang-migrate in the Go one. Both can run against one database (Flyway
  adopts it at version 2), but a new migration has to be added to both
  projects.

## Known limitations

These match the Go template, with the Java-specific items at the end.

- **Rate limits are per process.** With several replicas the effective limit
  is multiplied; move the buckets to a shared store (for example Redis) to
  scale out.
- **Rate limits key on the TCP peer address.** Behind a reverse proxy every
  client shares the proxy's IP. Derive the client IP from the proxy's header
  there, and trust that header only when the proxy is the sole way in.
- **The per-email limit can slow down a victim's logins.** An attacker who
  knows an email can keep its bucket empty.
- **No email verification, password reset, MFA or account deletion endpoints.**
- **Front end and API must be same-site.** The session cookie is
  `SameSite=Lax` and the refresh cookie `SameSite=Strict`, so browsers send
  them only when the page and the API share a registrable domain.
- **Access tokens cannot be revoked** before they expire. After logout, a role
  change or account deletion, a token already issued stays valid, with its old
  roles, for up to 15 minutes. Session mode reads roles on every request.
- **Roles in the demo** come from `users.role`; no route uses a role rule. Add
  `hasRole(...)` to a chain to use it.
- **HS256 with one shared secret.** Every service that verifies tokens can
  also mint them.
- **No metrics or tracing.**
- **Memory per hash.** Each Argon2id hash allocates 19 MiB on the Java heap,
  and up to `availableProcessors()` run at once. A burst of logins can need
  that many times 19 MiB of heap on top of the rest of the app, so size the
  container for it. The image sets `-XX:MaxRAMPercentage=70`; lower the CPU
  count of the container, or the semaphore size in
  [PasswordConfig](src/main/java/io/github/brianhuanggg/authkit/config/PasswordConfig.java),
  if memory is tight.
- **Tied to Spring Boot 4.** It uses Spring Security 7 (lambda DSL only) and
  Jackson 3 (`tools.jackson.*`). Boot 3 code and snippets that use the old DSL
  or `com.fasterxml.jackson` imports will not compile against it.
- **The front end is the Go template's React demo** and is not copied here.

## Layout

```
src/main/java/io/github/brianhuanggg/authkit/
  AuthkitApplication, Healthcheck, LogFormat   entry point, the `healthcheck` command, LOG_FORMAT
  config/        properties, security chains, data source, JWT and password beans
  auth/          login authentication, principal, user details
  password/      PepperedArgon2Encoder
  session/       session service, filter, memory and Postgres stores
  token/         JwtService, RefreshTokenService, memory and Postgres stores
  ratelimit/     Limiter, RateLimitFilter, client IP, Janitor
  user/          user repository
  web/           controllers, error handling, request log, body size, CSRF and cache filters
src/main/resources/db/migration/    Flyway migrations
src/test/                           JUnit 5, MockMvc, Testcontainers
Dockerfile, Makefile                image, and `make integration` against the Go suite
```
