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

docker-build: docker-build-service docker-build-server docker-build-static docker-build-kremlin-indexer docker-build-mil-indexer docker-build-mid-indexer

docker-build-service:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
    	--tag ${REGISTRY}/feed-parser-service:${IMAGE_TAG} \
    	--file ./Dockerfile .

docker-build-server:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
    	--tag ${REGISTRY}/feed-server:${IMAGE_TAG} \
    	--file ./Dockerfile_srv .

docker-build-static:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
    	--tag ${REGISTRY}/feed-static-generator:${IMAGE_TAG} \
    	--file ./Dockerfile_static .

docker-build-kremlin-indexer:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
    	--tag ${REGISTRY}/feed-kremlin-indexer:${IMAGE_TAG} \
    	--file ./Dockerfile_kremlin .

docker-build-mil-indexer:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
		--tag ${REGISTRY}/feed-mil-indexer:${IMAGE_TAG} \
		--file ./Dockerfile_mil .

docker-build-mid-indexer:
	DOCKER_BUILDKIT=1 docker --log-level=debug build --pull --build-arg BUILDKIT_INLINE_CACHE=1 \
		--tag ${REGISTRY}/feed-mid-indexer:${IMAGE_TAG} \
		--file ./Dockerfile_mid .

push:
	docker push ${REGISTRY}/feed-parser-service:${IMAGE_TAG}
	docker push ${REGISTRY}/feed-server:${IMAGE_TAG}
	docker push ${REGISTRY}/feed-static-generator:${IMAGE_TAG}
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
