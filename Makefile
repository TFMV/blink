.PHONY: build docker-build docker-push k8s-deploy k8s-delete

# Variables
IMAGE_NAME := blink
IMAGE_TAG := latest
DOCKER_REPO := your-docker-repo  # Change this to your Docker repository

build:
	go build -o blink ./cmd/blink

docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

docker-push:
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(DOCKER_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(DOCKER_REPO)/$(IMAGE_NAME):$(IMAGE_TAG)

k8s-deploy:
	kubectl apply -f kubernetes/configmap.yaml
	kubectl apply -f kubernetes/deployment.yaml
	kubectl apply -f kubernetes/service.yaml

k8s-delete:
	kubectl delete -f kubernetes/service.yaml
	kubectl delete -f kubernetes/deployment.yaml
	kubectl delete -f kubernetes/configmap.yaml 