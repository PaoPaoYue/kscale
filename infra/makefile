TIMEOUT := 100s
DATADOG_VERSION := 3.61.0

# remember to set the data-dog api key in datadog-secret before running this
# > helm repo add datadog https://helm.datadoghq.com
# > kubectl create secret generic datadog-secret --from-literal api-key=<DATADOG_API_KEY> --namespace datadog
datadog:
	helm upgrade --wait --timeout=$(TIMEOUT) --version ${DATADOG_VERSION} --create-namespace --namespace datadog --install --values datadog.yaml datadog-agent datadog/datadog


nfs:
	kubectl apply -f nfs.yaml --timeout=$(TIMEOUT)

rabbitmq:
	kubectl apply -f rabbitmq.yaml --timeout=$(TIMEOUT)

.PHONY: nfs rabbitmq