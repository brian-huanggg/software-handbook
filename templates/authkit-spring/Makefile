COMPOSE := docker compose

.DEFAULT_GOAL := help
.PHONY: help test package stack-up stack-down

help:
	@echo "targets: test package stack-up stack-down"

test:
	./mvnw test
package:
	./mvnw -DskipTests package

# JWT_SECRET (>= 32 bytes) comes from the environment, or is generated per run.
# Export your own to keep tokens valid across runs.
JWT_SECRET ?= $(shell openssl rand -hex 32)
export JWT_SECRET
stack-up:
	$(COMPOSE) up -d --build --wait
stack-down:
	$(COMPOSE) down
