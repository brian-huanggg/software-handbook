# authkit

A small, dependency-light authentication toolkit in Go, with a runnable demo
(API, Postgres, React front end). It implements two authentication modes side
by side so you can compare them and copy the one you need:

| | Session mode | Token mode |
|---|---|---|
| Credential | Opaque session ID in an HttpOnly cookie | Short-lived access JWT in the `Authorization` header |
| Server state | One row per session | One row per refresh token; access JWTs are stateless |
| Revocation | Immediate (delete the row) | Access JWT lives until it expires (15 min); refresh chain is revoked immediately |
| Endpoints | `/session/*` | `/token/*` |

Passwords are hashed with Argon2id, users live in Postgres, and roles are
checked by a small authorization layer.

> This is a learning and starting-point template, not a hosted product. Read
> [Known limitations](#known-limitations) before using it in production.

## Quick start

Requirements: Go 1.26+, Docker, `openssl`, and
[`golang-migrate`](https://github.com/golang-migrate/migrate) (only for local
runs outside Docker).

### Full stack in containers

```sh
make stack-up        # db, migrations, api (:8080), web (:3000)
open http://localhost:3000
make stack-down
```

`JWT_SECRET` is generated for each run. Export your own to keep it stable.

### API on the host

```sh
make up              # start Postgres only
make migrate-up
JWT_SECRET=$(openssl rand -hex 32) make run
```

### Tests

```sh
go test ./...        # Postgres-backed tests skip unless TEST_DATABASE_URL is set, e.g.
                     # TEST_DATABASE_URL="postgres://authkit:authkit@localhost:5432/authkit?sslmode=disable"
make integration     # starts the stack, then runs the end-to-end tests
make docs            # Swagger UI for api/openapi.yaml at http://localhost:8081
```

## Configuration

All settings come from environment variables. The server refuses to start if
the required ones are missing, so it never runs with an unsafe default.

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DATABASE_URL` | yes | – | Postgres connection string |
| `JWT_SECRET` | yes | – | HMAC key for access tokens, at least 32 bytes |
| `JWT_ISSUER` | no | `authkit` | `iss` claim, checked on verify |
| `PASSWORD_PEPPER` | no | none | Secret mixed into password hashes; keep it outside the database |
| `CORS_ORIGINS` | no | none | Comma-separated browser origins allowed to call the API. Each must be `scheme://host[:port]`, with no path and no `*` |
| `COOKIE_INSECURE` | no | off | `1` or `true` drops the `Secure` cookie flag. Development only |
| `ADDR` | no | `:8080` | Listen address |

Lifetimes are constants in [cmd/api/config.go](cmd/api/config.go): session 24 h,
refresh cookie 7 days. Access token TTL (15 min) and refresh family TTL (30
days) are package defaults.

Changing `PASSWORD_PEPPER` after users exist invalidates their stored hashes.
Treat it as fixed once chosen.

## HTTP API

The full contract is in [api/openapi.yaml](api/openapi.yaml). Summary:

| Method and path | Auth | Description |
|---|---|---|
| `GET /healthz` | none | Liveness check |
| `POST /register` | none | Create an account (`email`, `password` of 8–128 bytes) |
| `POST /session/login` | none | Verify credentials, set the `sid` cookie |
| `POST /session/logout` | cookie | Delete the session, clear the cookie |
| `GET /session/me` | cookie | Return `user_id` and `roles` |
| `POST /token/login` | none | Return an access JWT, set the `refresh_token` cookie |
| `POST /token/refresh` | refresh cookie | Rotate the refresh token, return a new access JWT |
| `POST /token/logout` | refresh cookie | Revoke the whole refresh chain |
| `GET /token/me` | bearer | Return `user_id` and `roles` |

Every `POST` must carry the header `X-Requested-With: XMLHttpRequest` (see
[CSRF](#csrf-and-cors)).

Example, token mode:

```sh
curl -s -X POST localhost:8080/register \
  -H 'Content-Type: application/json' -H 'X-Requested-With: XMLHttpRequest' \
  -d '{"email":"a@example.com","password":"correct horse battery"}'

curl -s -c jar -X POST localhost:8080/token/login \
  -H 'Content-Type: application/json' -H 'X-Requested-With: XMLHttpRequest' \
  -d '{"email":"a@example.com","password":"correct horse battery"}'
# {"access_token":"...","token_type":"Bearer","expires_in":900}

curl -s localhost:8080/token/me -H "Authorization: Bearer $ACCESS"
```

The refresh cookie is `Secure`, so plain-HTTP `curl` needs
`COOKIE_INSECURE=1` on the server.

## Using the packages

The packages under the module root are independent building blocks. The demo
in [cmd/api](cmd/api) is one way to assemble them.

| Package | Responsibility |
|---|---|
| `authn` | The `Principal` type, the `Authenticator` interface, and helpers to carry a principal in `context.Context` |
| `password` | Argon2id hashing and verification behind a `Hasher` interface |
| `session` | `Manager` (login, logout, authenticate) and a `Store` interface with memory and Postgres implementations |
| `token` | `Access` (JWT issue and verify), `Bearer` authenticator, `RefreshManager` and a `RefreshStore` interface with memory and Postgres implementations |
| `middleware` | `Require` (authenticate), `CORS`, `CSRF` |
| `authz` | `RequireRole` and `HasRole` |
| `store` | Postgres user store |

Protecting a route in your own service:

```go
sessions := session.NewManager(session.NewPostgresStore(pool), session.Options{
    Roles: rolesLookup, // func(ctx, userID) ([]string, error)
})

mux.Handle("GET /me", middleware.Require(sessions)(meHandler))
mux.Handle("GET /admin", middleware.Require(sessions)(
    authz.RequireRole("admin")(adminHandler)))

// inside a handler
p, _ := authn.FromContext(r.Context())
```

For token mode, replace `sessions` with `token.NewBearer(access)`. Anything
that implements `authn.Authenticator` plugs into `middleware.Require`, so you
can add your own scheme (API keys, mTLS) without touching the rest.

## Design

### Layering

```
HTTP handler  ->  middleware.Require  ->  authn.Authenticator  ->  Store
                       |                    (session.Manager
                       v                     or token.Bearer)
                 authn.Principal in ctx
                       |
                       v
                 authz.RequireRole
```

Authentication answers "who are you" and produces a `Principal`. Authorization
answers "may you do this" and runs after it. A missing or invalid credential
gives 401, a valid principal without the role gives 403, and an infrastructure
failure gives 500 with a generic body. `RequireRole` fails closed with 401 if
no principal is in the context, and panics at startup if called with no
roles.

Stores are small interfaces. Each has an in-memory implementation for tests
and a Postgres one, and both run the same shared suite
(`session/storetest`, `token/storetest`), so a new backend only needs to pass
that suite.

### Sessions

- The raw session ID is 32 random bytes and exists only in the client's
  cookie. The database stores its SHA-256 hash, so a leaked database cannot be
  replayed as live sessions.
- `Login` always issues a new ID and deletes any session the request already
  carried, which defeats session fixation.
- Stores do not enforce expiry. `Manager.Authenticate` does, and deletes an
  expired session when its cookie comes back.
- Roles are looked up on every request through the `Roles` callback, so a role
  change or a deleted account takes effect immediately.
- Cookie: `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`.

### Tokens

- **Access token**: HS256 JWT with `sub`, `iss`, `iat`, `exp` and `roles`.
  The algorithm is pinned in the parser, which blocks `alg=none` and algorithm
  confusion. The issuer and expiry are required. Every verification failure
  returns the same `ErrUnauthenticated`, so the cause is not leaked.
  Roles are baked into the token, so a role change takes effect on the next
  refresh (at most 15 minutes).
- **Refresh token**: 32 random bytes, stored only as a SHA-256 hash, delivered
  as an HttpOnly, `Secure`, `SameSite=Strict` cookie whose path is `/token`,
  so the browser never sends it to the rest of the API.
- **Rotation**: every refresh consumes the old token and issues a new one in a
  single atomic store operation. If it fails, the old token stays usable and
  the client can retry.
- **Reuse detection**: presenting an already-used token is treated as theft
  and revokes the whole family (every token descended from that login). A
  lost race between two concurrent refreshes is handled the same way.
- **Two lifetimes**: each token has an idle TTL (7 days in the demo), and the
  whole chain has an absolute TTL (30 days) fixed at login. Rotation never
  extends past it.

### Passwords

- Argon2id with the OWASP minimum parameters (19 MiB, 2 iterations, 1 thread),
  encoded as a PHC string. Parameters are stored in each hash, so you can raise
  them later and old hashes still verify.
- `Verify` caps the parameters it will accept from a stored hash, so a
  corrupted or tampered row cannot trigger huge allocations.
- An optional pepper is applied with HMAC-SHA256 before hashing.
- Login always performs exactly one hash verification. For an unknown email it
  verifies against a dummy hash, so response time and the error message do not
  reveal which accounts exist.
- Request bodies are limited to 64 KiB, strictly decoded (unknown fields are
  rejected), and passwords are limited to 8–128 bytes to bound hashing work.

### CSRF and CORS

Cookie-authenticated endpoints are protected by two independent checks on
every unsafe method (anything except `GET`, `HEAD`, `OPTIONS`):

1. The `X-Requested-With: XMLHttpRequest` header is required. Cross-site pages
   cannot add custom headers without a CORS preflight, and HTML forms cannot
   set it at all.
2. If the request has an `Origin`, it must be in the allowlist or match the
   API's own host.

CORS is an explicit allowlist with credentials. No wildcard and no echoing of
arbitrary origins. Origins that are not listed receive no CORS headers. CORS
runs outermost so preflights are answered before the CSRF check. Because
`GET` is not checked, `GET` handlers must never change state.

### Database

Migrations are in [migrations/](migrations). Tables:

- `users`: UUID key, case-insensitive unique email (`citext`) with a format
  check, `password_hash`, `role` (`user` or `admin`), timestamps.
- `sessions`: `id_hash`, `user_id`, `expires_at`.
- `refresh_tokens`: `id_hash`, `user_id`, `family_id`, `used_at`,
  `expires_at`, `family_expires_at`.

Sessions and refresh tokens cascade when a user is deleted.

### Front end

[web](web) is a React + Vite + shadcn login page served by nginx on its own
origin (`:3000`), so the demo exercises real cross-origin cookie and CORS
rules. It sends a strict Content-Security-Policy.
[web/src/lib/api.ts](web/src/lib/api.ts) shows the client side of both modes:
the access token is kept only in memory (never `localStorage`), and concurrent
requests share a single refresh call, because a second parallel refresh would
look like token reuse and log the user out.

### Deployment image

The API builds into a distroless, non-root image. It has no shell, so the
Compose health check runs `/api healthcheck`.

## Known limitations

- **No scheduled cleanup.** `session.Manager.Purge` and
  `token.RefreshManager.Purge` exist but nothing calls them. Run them
  periodically (a goroutine or a cron job), or expired rows will accumulate.
- **No rate limiting or lockout** on login or register. Add it at a gateway or
  as middleware.
- **No email verification, password reset, MFA or account deletion endpoints.**
- **Access tokens cannot be revoked** before they expire. Keep the TTL short.
- **Roles in the demo** come from the `users.role` column and nothing in the
  demo routes uses `authz.RequireRole`. It is there for you to apply.
- **HS256 with one shared secret.** Every service that verifies tokens can
  also mint them. Move to asymmetric keys if other services need to verify.
- **No request logging, metrics or tracing.**

## Layout

```
cmd/api/       server entry point, config, handlers, health check
authn/         Principal and Authenticator
authz/         role checks
password/      Argon2id hasher
session/       session manager, memory and Postgres stores, shared store tests
token/         access JWT, bearer authenticator, refresh manager, stores
middleware/    Require, CORS, CSRF
store/         Postgres user store
migrations/    SQL schema
api/           OpenAPI spec
web/           React demo front end
integration/   end-to-end tests (build tag: integration)
```
