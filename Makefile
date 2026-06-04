.PHONY: up seed logs down clean backend-test

up:
	docker compose up -d --build

seed:
	docker compose exec -T backend /app/server seed

logs:
	docker compose logs -f

down:
	docker compose down

clean:
	docker compose down -v

backend-test:
	cd backend && go test ./...
