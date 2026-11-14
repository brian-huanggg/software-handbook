# Runs the Go template's end-to-end suite, unchanged, against this API.
GO_DIR := ../authkit
COMPOSE := docker compose -f $(GO_DIR)/docker-compose.yml --profile java
API_JAVA_PORT ?= 8082

.DEFAULT_GOAL := help
.PHONY: help test package stack-up stack-down integration

help:
	@echo "targets: test package stack-up stack-down integration"

test:
	./mvnw test
package:
	./mvnw -DskipTests package

# JWT_SECRET (>= 32 bytes) comes from the environment, or is generated per run.
JWT_SECRET ?= $(shell openssl rand -hex 32)
export JWT_SECRET
export API_JAVA_PORT
stack-up:
	$(COMPOSE) up -d --build --wait db api-java
stack-down:
	$(COMPOSE) down
# The suite logs in dozens of times from one IP; raise the limiter bursts so
# it measures auth behaviour, not the rate limiter (unit-tested separately).
integration: export RATE_LIMIT_IP_BURST = 1000
integration: export RATE_LIMIT_EMAIL_BURST = 1000
integration: stack-up
	cd $(GO_DIR) && API_URL=http://localhost:$(API_JAVA_PORT) go test -tags integration -count=1 -v ./integration/
