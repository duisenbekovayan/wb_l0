APP_NAME=wb-orders
DB_URL=postgres://wb:wb@localhost:5432/wb_orders?sslmode=disable

# go
run:
	go run ./cmd/app

tidy:
	go mod tidy

build:
	go build -o bin/$(APP_NAME) ./cmd/app

# docker
up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

# migrations (golang-migrate required!)
migrate-up:
	migrate -path ./migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DB_URL)" down 1

migrate-force:
	migrate -path ./migrations -database "$(DB_URL)" force 1

migrate-version:
	migrate -path ./migrations -database "$(DB_URL)" version
