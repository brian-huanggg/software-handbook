DB_URL=postgres://authkit:authkit@localhost:5432/authkit?sslmode=disable

up:
	docker compose up -d db
migrate-up:
	migrate -path migrations -database "$(DB_URL)" up
migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1
