init: init-ci frontend-ready
init-ci: frontend-clear \
	docker-pull docker-build docker-up \
	frontend-init
up: docker-up
down: docker-down
restart: down up

docker-up:
	docker compose up -d
docker-down:
	docker compose down --remove-orphans

frontend-init: frontend-yarn-install

frontend-yarn-install:
	docker-compose run --rm frontend-node-cli yarn install

frontend-ready:
	docker run --rm -v ${PWD}/frontend:/app -w /app alpine touch .ready

frontend-clear:
	docker run --rm -v ${PWD}/frontend:/app -w /app alpine sh -c 'rm -rf .ready build'

docker-down-clear:
	docker compose down -v --remove-orphans

docker-pull:
	docker compose pull

docker-build:
	docker compose build --pull
