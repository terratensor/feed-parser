init: init-ci
init-ci:
	docker-pull docker-build docker-up
up: docker-up
down: docker-down
restart: down up

docker-up:
	docker compose up -d
docker-down:
	docker compose down --remove-orphans

docker-down-clear:
	docker compose down -v --remove-orphans

docker-pull:
	docker compose pull

docker-build:
	docker compose build --pull

dev-docker-build:
	REGISTRY=localhost IMAGE_TAG=main-1 make docker-build

docker-build:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
		--target builder \
		--cache-from ${REGISTRY}/feed-parser-service:cache-builder \
		--tag ${REGISTRY}/feed-parser-service:cache-builder \
		--file ./Dockerfile .

	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
    	--cache-from ${REGISTRY}/feed-parser-service:cache-builder \
    	--cache-from ${REGISTRY}/feed-parser-service:cache \
    	--tag ${REGISTRY}/feed-parser-service:cache \
    	--tag ${REGISTRY}/feed-parser-service:${IMAGE_TAG} \
    	--file ./Dockerfile .

push-build-cache:
	docker push ${REGISTRY}/feed-parser-service:cache-builder
	docker push ${REGISTRY}/feed-parser-service:cache

push:
	docker push ${REGISTRY}/feed-parser-service:${IMAGE_TAG}

deploy:
	ssh -o StrictHostKeyChecking=no deploy@${HOST} -p ${PORT} 'docker network create --driver=overlay traefik-public || true'
	ssh -o StrictHostKeyChecking=no deploy@${HOST} -p ${PORT} 'rm -rf feed-parser-service_${BUILD_NUMBER} && mkdir feed-parser-service_${BUILD_NUMBER}'

	envsubst < docker-compose-production.yml > docker-compose-production-env.yml
	scp -o StrictHostKeyChecking=no -P ${PORT} docker-compose-production-env.yml deploy@${HOST}:feed-parser-service_${BUILD_NUMBER}/docker-compose.yml
	rm -f docker-compose-production-env.yml

	ssh -o StrictHostKeyChecking=no deploy@${HOST} -p ${PORT} 'cd feed-parser-service_${BUILD_NUMBER} && docker stack deploy --compose-file docker-compose.yml feed-parser-service --with-registry-auth --prune'
