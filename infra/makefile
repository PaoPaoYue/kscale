TIMEOUT := 100s
DATADOG_VERSION := 3.61.0
NVIDIA_DEVICE_PLUGIN_VERSION := 0.17.1
KUBERAY_VERSION := 1.3.0

calico:
# restart calico-node and calico-kube-controllers if they are already running
# kubectl -n kube-system rollout restart daemonset calico-node
# kubectl -n kube-system rollout restart deployment calico-kube-controllers
	kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml --timeout=$(TIMEOUT)

# remember to set the data-dog api key in datadog-secret before running this
# > helm repo add datadog https://helm.datadoghq.com
# > kubectl create secret generic datadog-secret --from-literal api-key=<DATADOG_API_KEY> --namespace datadog
datadog:
	helm upgrade --wait --timeout=$(TIMEOUT) --version ${DATADOG_VERSION} --create-namespace --namespace datadog --install --values datadog.yaml datadog-agent datadog/datadog

# kubectl create namepsace ypp
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

# helm repo add gpu-helm-charts https://nvidia.github.io/dcgm-exporter/helm-charts
# helm repo update
dcgm:
	kubectl apply -f dcgm-config.yaml --timeout=$(TIMEOUT)
	helm upgrade -i dcgm-datadog gpu-helm-charts/dcgm-exporter -n datadog -f dcgm-values.yaml --timeout=$(TIMEOUT)

# helm repo add kuberay https://ray-project.github.io/kuberay-helm/
# helm repo update
kuberay:
	helm upgrade --install kuberay-operator kuberay/kuberay-operator \
	--namespace kuberay \
	--create-namespace \
	--version $(KUBERAY_VERSION) \
	--values kuberay.yaml \
	--timeout=$(TIMEOUT)

.PHONY: calico datadog nfs nvidia-device-plugin kuberay