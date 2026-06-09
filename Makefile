.PHONY: up test dev prod seed logs down clean backend-test

up:
	docker compose up -d --build

test dev:
	DEV_AUTH_ENABLED=true VITE_DEV_AUTH=true docker compose up -d --build
	$(MAKE) seed

prod:
	DEV_AUTH_ENABLED=false VITE_DEV_AUTH=false docker compose up -d --build
	$(MAKE) seed

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
