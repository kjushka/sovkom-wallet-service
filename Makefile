run:
	docker compose build --no-cache
	docker compose up -d
	docker compose logs -f

stop:
	docker compose stop

down:
	docker compose down

logs:
	docker compose logs -f

restart: down run