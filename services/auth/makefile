swagger-init:
	swag init -g cmd/app/main.go --parseDependency --parseInternal --parseDepth 3
test-db-up:
	docker compose -f docker-compose-test.yaml down --remove-orphans && \
 	docker compose --env-file  ./cmd/app/.env.test -f docker-compose-test.yaml up
test-db-rebuild:
	docker compose -f docker-compose-test.yaml down --remove-orphans && \
 	docker compose -f docker-compose-test.yaml build --no-cache && \
 	docker compose --env-file  ./cmd/app/.env.test -f docker-compose-test.yaml up
app-up:
	docker compose -f docker-compose.yaml down && \
 	docker compose --env-file .env -f docker-compose.yaml up
app-rebuild:
	docker compose -f docker-compose.yaml down && \
 	docker compose -f docker-compose.yaml build --no-cache && \
 	docker compose --env-file  .env -f docker-compose.yaml up
