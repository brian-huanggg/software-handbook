.PHONY: up down pg mysql bench-pg bench-mysql bench test

up:
	docker compose up -d --wait

down:
	docker compose down -v

pg:
	docker compose exec postgres psql -U postgres -d demo

mysql:
	docker compose exec mysql mysql -uroot -pdemo demo

bench-pg:
	docker compose exec -T postgres psql -U postgres -d demo -v ON_ERROR_STOP=1 < sql/pg_bench.sql

bench-mysql:
	docker compose exec -T mysql mysql -uroot -pdemo demo < sql/mysql_bench.sql

bench: up bench-pg bench-mysql

test:
	go vet ./... && go test ./...
