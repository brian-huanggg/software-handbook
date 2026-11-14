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
run:
	DATABASE_URL="$(DB_URL)" go run ./cmd/api
