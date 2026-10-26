.PHONY: up down logs run test test-unit test-integration lint

up:                
	docker compose up -d --build
down:           
	docker compose down
logs:
	docker compose logs -f app
run:              
	go run ./cmd/taskmanager
test-unit:        
	go test -count=1 ./internal/domain/...
test-integration: 
	go test -count=1 -v ./internal/integration/...
test: test-unit test-integration
lint:
	go vet ./...
