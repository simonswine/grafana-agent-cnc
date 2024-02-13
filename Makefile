REGISTRY := simonswine/grafana-agent-cnc
TAG := next
PLATFORM := linux/amd64,linux/arm64
KIND_CLUSTER := agents

export HELM_REPOSITORY_CONFIG := $(CURDIR)/.helm.repo.yaml

.PHONY: image-build
image-build:
	docker buildx build --load --platform $(PLATFORM) -t $(REGISTRY):$(TAG) .

.PHONY: image-push
image-push:
	docker buildx build --push --platform $(PLATFORM) -t $(REGISTRY):$(TAG) .

.PHONY: cluster
cluster: 
	kind export kubeconfig --name $(KIND_CLUSTER) || kind create cluster --name $(KIND_CLUSTER) --config operations/kind.yaml

.PHONY: cluster-delete
cluster-delete:
	kind delete cluster --name $(KIND_CLUSTER)

.PHONY: demo
demo:
	helm upgrade --install pyroscope ./operations/helm/grafana-agent-cnc/
	helm repo add jenkins https://charts.jenkins.io
	helm repo add k8s-at-home https://k8s-at-home.com/charts/
	helm repo add gabe565 https://charts.gabe565.com
	helm repo update
	# java example
	helm upgrade --install -n jenkins --create-namespace jenkins jenkins/jenkins
	# dotnet example
	helm upgrade --install -n lidarr --create-namespace lidarr k8s-at-home/lidarr
	# python example
	helm upgrade --install -n healthchecks --create-namespace healthchecks gabe565/healthchecks

.PHONY: demo-delete
demo-delete:
	helm delete --ignore-not-found pyroscope
	helm delete --ignore-not-found -n jenkins jenkins
	helm delete --ignore-not-found -n lidarr lidarr
	helm delete --ignore-not-found -n healthchecks healthchecks
