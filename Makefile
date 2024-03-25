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

docker-build: docker-build-service docker-build-kremlin-indexer docker-build-mil-indexer docker-build-mid-indexer

docker-build-service:
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

docker-build-kremlin-indexer:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
		--target builder \
		--cache-from ${REGISTRY}/feed-kremlin-indexer:cache-builder \
		--tag ${REGISTRY}/feed-kremlin-indexer:cache-builder \
		--file ./Dockerfile_kremlin .

	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
    	--cache-from ${REGISTRY}/feed-kremlin-indexer:cache-builder \
    	--cache-from ${REGISTRY}/feed-kremlin-indexer:cache \
    	--tag ${REGISTRY}/feed-kremlin-indexer:cache \
    	--tag ${REGISTRY}/feed-kremlin-indexer:${IMAGE_TAG} \
    	--file ./Dockerfile_kremlin .

docker-build-mil-indexer:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
    		--target builder \
    		--cache-from ${REGISTRY}/feed-mil-indexer:cache-builder \
    		--tag ${REGISTRY}/feed-mil-indexer:cache-builder \
    		--file ./Dockerfile_mil .

	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
		--cache-from ${REGISTRY}/feed-mil-indexer:cache-builder \
		--cache-from ${REGISTRY}/feed-mil-indexer:cache \
		--tag ${REGISTRY}/feed-mil-indexer:cache \
		--tag ${REGISTRY}/feed-mil-indexer:${IMAGE_TAG} \
		--file ./Dockerfile_mil .

docker-build-mid-indexer:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
    		--target builder \
    		--cache-from ${REGISTRY}/feed-mid-indexer:cache-builder \
    		--tag ${REGISTRY}/feed-mid-indexer:cache-builder \
    		--file ./Dockerfile_mid .

	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
		--cache-from ${REGISTRY}/feed-mid-indexer:cache-builder \
		--cache-from ${REGISTRY}/feed-mid-indexer:cache \
		--tag ${REGISTRY}/feed-mid-indexer:cache \
		--tag ${REGISTRY}/feed-mid-indexer:${IMAGE_TAG} \
		--file ./Dockerfile_mid .

push-build-cache: push-build-cache-service push-build-cache-kremlin-indexer push-build-cache-mil-indexer push-build-cache-mid-indexer

push-build-cache-service:
	docker push ${REGISTRY}/feed-parser-service:cache-builder
	docker push ${REGISTRY}/feed-parser-service:cache

push-build-cache-kremlin-indexer:
	docker push ${REGISTRY}/feed-kremlin-indexer:cache-builder
	docker push ${REGISTRY}/feed-kremlin-indexer:cache

push-build-cache-mil-indexer:
	docker push ${REGISTRY}/feed-mil-indexer:cache-builder
	docker push ${REGISTRY}/feed-mil-indexer:cache

push-build-cache-mid-indexer:
	docker push ${REGISTRY}/feed-mid-indexer:cache-builder
	docker push ${REGISTRY}/feed-mid-indexer:cache

push:
	docker push ${REGISTRY}/feed-parser-service:${IMAGE_TAG}
	docker push ${REGISTRY}/feed-kremlin-indexer:${IMAGE_TAG}
	docker push ${REGISTRY}/feed-mil-indexer:${IMAGE_TAG}
	docker push ${REGISTRY}/feed-mid-indexer:${IMAGE_TAG}

deploy:
	ssh -o StrictHostKeyChecking=no deploy@${HOST} -p ${PORT} 'docker network create --driver=overlay traefik-public || true'
	ssh -o StrictHostKeyChecking=no deploy@${HOST} -p ${PORT} 'docker network create --driver=overlay feed-parser-net || true'
	ssh -o StrictHostKeyChecking=no deploy@${HOST} -p ${PORT} 'rm -rf feed-parser-service_${BUILD_NUMBER} && mkdir feed-parser-service_${BUILD_NUMBER}'

	envsubst < docker-compose-production.yml > docker-compose-production-env.yml
	scp -o StrictHostKeyChecking=no -P ${PORT} docker-compose-production-env.yml deploy@${HOST}:feed-parser-service_${BUILD_NUMBER}/docker-compose.yml
	rm -f docker-compose-production-env.yml

	ssh -o StrictHostKeyChecking=no deploy@${HOST} -p ${PORT} 'cd feed-parser-service_${BUILD_NUMBER} && docker stack deploy --compose-file docker-compose.yml feed-parser-service --with-registry-auth --prune'
