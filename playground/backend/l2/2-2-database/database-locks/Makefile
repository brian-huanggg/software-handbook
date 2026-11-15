.PHONY: up down demo test exercise

up:
	docker compose up -d --wait

down:
	docker compose down -v

demo:
	go run ./cmd/demo

# 連不上資料庫時整合測試會自動略過。
test:
	go vet ./... && go test ./...

exercise:
	go test -tags exercise ./exercise
