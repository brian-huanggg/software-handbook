DB_URL=postgres://authkit:authkit@localhost:5432/authkit?sslmode=disable

up:
	docker compose up -d db
migrate-up:
	migrate -path migrations -database "$(DB_URL)" up
migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1
# Needs JWT_SECRET (>= 32 bytes) in the environment, e.g.
#   JWT_SECRET=$(openssl rand -hex 32) make run
# Set COOKIE_INSECURE=1 to test cookies over plain HTTP with non-Chromium tools.
# CORS_ORIGINS is the comma-separated list of browser origins allowed to call
# the API; it defaults to the web container on :3000.
CORS_ORIGINS ?= http://localhost:3000
run:
	DATABASE_URL="$(DB_URL)" CORS_ORIGINS="$(CORS_ORIGINS)" go run ./cmd/api

# Swagger UI for api/openapi.yaml at http://localhost:8081. "Try it out" from
# this origin needs it in CORS_ORIGINS, e.g.
#   CORS_ORIGINS=http://localhost:8081 make run
docs:
	docker run --rm -p 8081:8080 -e SWAGGER_JSON=/spec/openapi.yaml \
		-v "$(CURDIR)/api:/spec" swaggerapi/swagger-ui
