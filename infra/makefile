TIMEOUT := 100s
DATADOG_VERSION := 3.61.0
NVIDIA_DEVICE_PLUGIN_VERSION := 0.17.1

# remember to set the data-dog api key in datadog-secret before running this
# > helm repo add datadog https://helm.datadoghq.com
# > kubectl create secret generic datadog-secret --from-literal api-key=<DATADOG_API_KEY> --namespace datadog
datadog:
	helm upgrade --wait --timeout=$(TIMEOUT) --version ${DATADOG_VERSION} --create-namespace --namespace datadog --install --values datadog.yaml datadog-agent datadog/datadog

nfs:
	kubectl apply -f nfs.yaml --timeout=$(TIMEOUT)

# helm repo add nvdp https://nvidia.github.io/k8s-device-plugin
# helm repo update
# helm search repo nvdp --devel
nvidia-device-plugin:
	helm upgrade -i nvdp nvdp/nvidia-device-plugin \
	--values nvidia-device-plugin.yaml \
	--namespace nvidia-device-plugin \
	--create-namespace \
	--version $(NVIDIA_DEVICE_PLUGIN_VERSION) \
	--timeout=$(TIMEOUT)

sd-webui:
	kubectl apply -f sdwebui-plain.yaml --timeout=$(TIMEOUT)


.PHONY: datadog nfs rabbitmq nvidia-device-plugin sd-webui