SERVICES = user-service order-service payment-service product-service notification-service
DOCKER_USER = jonathangunawan30

.PHONY: build push deploy all build-ps push-ps deploy-ps all-ps

build:
	@for svc in $(SERVICES); do \
		echo "Building $$svc..."; \
		docker build -t $(DOCKER_USER)/$$svc ./$$svc; \
	done

push:
	@for svc in $(SERVICES); do \
		echo "Pushing $$svc..."; \
		docker push $(DOCKER_USER)/$$svc; \
	done

deploy:
	@for svc in $(SERVICES); do \
		docker pull $(DOCKER_USER)/$$svc; \
	done
	docker compose up -d --force-recreate

all: build push

build-ps:
	powershell -Command "foreach ($$svc in @('user-service','order-service','payment-service','product-service','notification-service')) { Write-Host \"Building $$svc...\"; docker build -t $(DOCKER_USER)/$$svc ./$$svc }"

push-ps:
	powershell -Command "foreach ($$svc in @('user-service','order-service','payment-service','product-service','notification-service')) { Write-Host \"Pushing $$svc...\"; docker push $(DOCKER_USER)/$$svc }"

deploy-ps:
	powershell -Command "foreach ($$svc in @('user-service','order-service','payment-service','product-service','notification-service')) { docker pull $(DOCKER_USER)/$$svc }"; docker compose up -d --force-recreate

all-ps: build-ps push-ps