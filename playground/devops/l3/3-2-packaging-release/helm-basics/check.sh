#!/usr/bin/env bash
# 不需要叢集：用 helm template 在本機渲染，檢查 values 真的改變了清單。
set -euo pipefail

CHART_VERSION="6.7.1"
REPO_URL="https://stefanprodan.github.io/podinfo"
cd "$(dirname "$0")"

helm repo add podinfo "$REPO_URL" --force-update >/dev/null

render() {
  helm template web podinfo/podinfo --version "$CHART_VERSION" "$@"
}

expect() {
  local name=$1 pattern=$2 manifest=$3
  if ! grep -Eq "$pattern" <<<"$manifest"; then
    echo "FAIL: $name (找不到 $pattern)" >&2
    exit 1
  fi
  echo "ok:   $name"
}

dev=$(render -f values-dev.yaml)
prod=$(render -f values-prod.yaml)
override=$(render -f values-prod.yaml --set replicaCount=5)

expect "dev 一個副本" '^  replicas: 1$' "$dev"
expect "prod 三個副本" '^  replicas: 3$' "$prod"
expect "prod 設了記憶體上限" 'memory: 128Mi' "$prod"
expect "--set 蓋過 -f" '^  replicas: 5$' "$override"

echo "all checks passed"
