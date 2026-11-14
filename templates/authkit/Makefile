DB_URL=postgres://authkit:authkit@localhost:5432/authkit?sslmode=disable

.DEFAULT_GOAL := help
.PHONY: help up migrate-up migrate-down stack-up stack-down integration run docs test vet

help:
	@echo "targets: test vet up migrate-up run stack-up stack-down integration docs"

test:
	go test ./...
vet:
	go vet ./...

up:
	docker compose up -d db
migrate-up:
	migrate -path migrations -database "$(DB_URL)" up
migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1
# JWT_SECRET (>= 32 bytes) comes from the environment if set; otherwise one is
# generated once into the git-ignored .jwt_secret and reused, so tokens survive
# restarts. Delete the file to rotate it. To use your own:
#   JWT_SECRET=... make run
# Set COOKIE_INSECURE=1 to test cookies over plain HTTP with non-Chromium tools.
# CORS_ORIGINS is the comma-separated list of browser origins allowed to call
# the API; it defaults to the web container on :3000.
CORS_ORIGINS ?= http://localhost:3000

# Full stack in containers (db, migrations, api, web) and the end-to-end test.
# LOG_FORMAT=json|text selects the log format (default text for `make run`).
ifndef JWT_SECRET
JWT_SECRET := $(shell test -s .jwt_secret || (umask 077; openssl rand -hex 32 > .jwt_secret); cat .jwt_secret)
endif
export JWT_SECRET
stack-up:
	docker compose up -d --build --wait db api web
stack-down:
	docker compose down
# The suite logs in dozens of times from one IP; raise the limiter bursts so
# it measures auth behaviour, not the rate limiter (unit-tested separately).
integration: export RATE_LIMIT_IP_BURST = 1000
integration: export RATE_LIMIT_EMAIL_BURST = 1000
integration: stack-up
	go test -tags integration -count=1 -v ./integration/

run:
	DATABASE_URL="$(DB_URL)" CORS_ORIGINS="$(CORS_ORIGINS)" go run ./cmd/api

# Swagger UI for api/openapi.yaml at http://localhost:8081. "Try it out" from
# this origin needs it in CORS_ORIGINS, e.g.
#   CORS_ORIGINS=http://localhost:8081 make run
docs:
	docker run --rm -p 8081:8080 -e SWAGGER_JSON=/spec/openapi.yaml \
		-v "$(CURDIR)/api:/spec" swaggerapi/swagger-ui
