VERSION_AUTH ?= v1
VERSION_BUDGET ?= v1
VERSION_NOTIFICATION ?= v1
VERSION_DOCS ?= v1
REPLICAS_AUTH ?= 3
REPLICAS_BUDGET ?= 3
REPLICAS_NOTIFICATION ?= 3
REPLICAS_DOCS ?= 1
LOCAL_PATH_VOLUMES ?= /home/dmitriy/volumes/minikube-persistent-disks #!EXAMPLE!
TAIL ?=1000
build-auth:
	docker build -t dmitriysyworov/auth-user-service:$(VERSION_AUTH) -f ./services/auth/Dockerfile . && \
	docker push dmitriysyworov/auth-user-service:$(VERSION_AUTH)

build-notification:
	docker build -t dmitriysyworov/notification:$(VERSION_NOTIFICATION) -f ./services/notification/Dockerfile . && \
	docker push dmitriysyworov/notification:$(VERSION_NOTIFICATION)

build-docs:
	docker build -t dmitriysyworov/gateway-docs:$(VERSION_DOCS) -f ./services/gatewaydocs/Dockerfile . && \
	docker push dmitriysyworov/gateway-docs:$(VERSION_DOCS)

build-all-images:
	@$(MAKE) -j 4 build-auth build-notification build-docs

rebuild-push-all-helm-hard-replace-all: build-all-images
#Delete
	-helm uninstall my --namespace infrastructure
	-minikube addons disable ingress
	-kubectl get pods -A --no-headers | awk '{print $$1, $$2}' | xargs -L1 sh -c 'kubectl patch pod $$2 -n $$1 -p "{\"metadata\":{\"finalizers\":null}}" --type=merge'
	-kubectl get pvc -A --no-headers | awk '{print $$1, $$2}' | xargs -L1 sh -c 'kubectl patch pvc $$2 -n $$1 -p "{\"metadata\":{\"finalizers\":null}}" --type=merge'
	-kubectl delete namespace infrastructure app --force --grace-period=0
	-kubectl get pv --no-headers | awk '{print $$1}' | xargs -I {} kubectl patch pv {} -p "{\"metadata\":{\"finalizers\":null}}" --type=merge
	-kubectl delete pv --all --force --grace-period=0
	-kubectl wait --for=delete namespace/app --timeout=60s
	-kubectl wait --for=delete namespace/infrastructure --timeout=60s
#Install Infrastructure
	helm install my oci://registry-1.docker.io/soldevelo/kafka-chart --version 32.4.4 \
 	-f ./infra-chart/values.yaml \
 	--namespace infrastructure \
 	--create-namespace
	kubectl wait --namespace infrastructure --for=condition=Ready pod/my-kafka-chart-controller-0 --timeout=180s
	sleep 20
	helm dependency build ./infra-chart
	helm install infrastructure ./infra-chart \
	-f ./infra-chart/values.yaml \
 	--namespace infrastructure
	kubectl wait --namespace infrastructure --for=condition=complete job -l app=kafka-topics-setup --timeout=180s
#Install Nginx
	minikube addons enable ingress
	kubectl wait --namespace ingress-nginx \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller \
		--timeout=120s
#Job
	helm dependency build ./app-chart
	helm install app ./app-chart \
		-f ./app-chart/values.yaml \
		--namespace app \
		--create-namespace \
		--set versions.authUserVersion="$(VERSION_AUTH)" \
		--set versions.budgetPlannerVersion="$(VERSION_BUDGET)" \
		--set versions.notificationVersion="$(VERSION_NOTIFICATION)" \
		--set versions.gatewayDocsVersion="$(VERSION_DOCS)" \
		--set replicasCount.authReplicas=0 \
		--set replicasCount.budgetReplicas=0 \
		--set replicasCount.notificationReplicas=0 \
		--set replicasCount.docsReplicas=0
	kubectl wait --namespace app --for=condition=complete job --all --timeout=320s
	kubectl delete job --all --namespace app --force --grace-period=0
#Final services
	helm upgrade app ./app-chart \
		-f ./app-chart/values.yaml \
		--namespace app \
		--reuse-values \
		--set versions.authUserVersion="$(VERSION_AUTH)" \
		--set versions.budgetPlannerVersion="$(VERSION_BUDGET)" \
		--set versions.notificationVersion="$(VERSION_NOTIFICATION)" \
		--set versions.gatewayDocsVersion="$(VERSION_DOCS)" \
		--set replicasCount.authReplicas="$(REPLICAS_AUTH)" \
		--set replicasCount.budgetReplicas="$(REPLICAS_BUDGET)" \
		--set replicasCount.notificationReplicas="$(REPLICAS_NOTIFICATION)" \
		--set replicasCount.docsReplicas="$(REPLICAS_DOCS)"
#Install ingress
	helm template app ./app-chart -f ./app-chart/values.yaml --show-only templates/ingress.yaml | kubectl apply --namespace app -f -

upgrade-helm-push-all: build-all-images
	helm upgrade app ./app-chart \
		-f ./app-chart/values.yaml \
		--namespace app \
		--set replicasCount.authReplicas="$(REPLICAS_AUTH)" \
		--set replicasCount.budgetReplicas="$(REPLICAS_BUDGET)" \
		--set replicasCount.notificationReplicas="$(REPLICAS_NOTIFICATION)" \
		--set replicasCount.docsReplicas="$(REPLICAS_DOCS)" \
		--set versions.authUserVersion="$(VERSION_AUTH)" \
        --set versions.budgetPlannerVersion="$(VERSION_BUDGET)" \
        --set versions.notificationVersion="$(VERSION_NOTIFICATION)" \
        --set versions.gatewayDocsVersion="$(VERSION_DOCS)"
	helm template app ./app-chart -f ./app-chart/values.yaml --show-only templates/ingress.yaml | kubectl apply --namespace app -f -
minikube-start-local:
	minikube start --cpus=4 --memory=8192 --driver=docker --mount --mount-string="$(LOCAL_PATH_VOLUMES):/mnt/data"
get-services-port:
	minikube service ingress-nginx-controller --namespace=ingress-nginx
get-logs-auth-user:
	kubectl logs -l app=app-auth-user --tail=$(TAIL) -f -n app
get-logs-notification:
	kubectl logs -l app=notification --tail=$(TAIL) -f -n app
get-logs-docs:
	kubectl logs -l app=gateway-docs --tail=$(TAIL) -f -n app
proto-update-all:
	protoc --go_out=. --go_opt=paths=source_relative ./shared/shprotos/event/user.proto
	protoc --go_out=. --go_opt=paths=source_relative ./shared/shprotos/event/letter.proto
check-template-infra:
	helm dependency build ./infra-chart
	helm template ./infra-chart -f ./infra-chart/values.yaml
check-template-app:
	helm dependency build ./app-chart
	helm template ./app-chart -f ./app-chart/values.yaml