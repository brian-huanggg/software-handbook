# Helm 基礎範例

用公開的 podinfo chart 練習 chart、values、release。需要 Helm 3；步驟 1 到 2 不需要叢集。

## 不需要叢集

```bash
./check.sh
```

## 需要叢集（任何本機 Kubernetes 皆可，例如 Docker Desktop、kind、minikube）

```bash
helm repo add podinfo https://stefanprodan.github.io/podinfo
helm install web podinfo/podinfo --version 6.7.1 -f values-dev.yaml -n demo --create-namespace
helm upgrade web podinfo/podinfo --version 6.7.1 -f values-prod.yaml -n demo
helm history web -n demo
helm rollback web 1 -n demo
helm uninstall web -n demo
```
