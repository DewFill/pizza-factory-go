up:
	docker-compose -f compose.yml -p pizza-factory-go up -d --build
down:
	docker compose down
clear:
	docker compose down -v --remove-orphans
	docker compose rm -vsf
watch:
	docker compose watch