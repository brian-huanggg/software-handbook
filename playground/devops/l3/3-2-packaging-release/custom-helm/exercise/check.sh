#!/usr/bin/env bash
# 不需要叢集：用 helm lint 與 helm template 檢查 chart。
# 用法：./check.sh [chart 目錄]，預設檢查 ./starter。
set -euo pipefail

cd "$(dirname "$0")"
chart=${1:-starter}
fail=0

check() {
  local name=$1 pattern=$2 manifest=$3
  if grep -Eq "$pattern" <<<"$manifest"; then
    echo "ok:   $name"
  else
    echo "FAIL: $name (找不到 $pattern)"
    fail=1
  fi
}

helm lint "$chart" -f "$chart/values-dev.yaml" >/dev/null
echo "ok:   helm lint"

dev=$(helm template web "$chart" -f "$chart/values-dev.yaml")
prod=$(helm template web "$chart" -f "$chart/values-prod.yaml")
nomig=$(helm template web "$chart" -f "$chart/values-dev.yaml" --set migration.enabled=false)

check "checksum/config annotation" 'checksum/config: [0-9a-f]{64}' "$dev"
check "prod 有 requests／limits 區塊" 'limits:' "$prod"
check "migration Job 存在" 'kind: Job' "$dev"
check "migration 是 pre-install,pre-upgrade hook" '"helm.sh/hook": pre-install,pre-upgrade' "$dev"
check "hook 結束後清掉舊 Job" 'before-hook-creation' "$dev"

if grep -q 'kind: Job' <<<"$nomig"; then
  echo "FAIL: migration.enabled=false 時不該有 Job"
  fail=1
else
  echo "ok:   migration.enabled=false 時沒有 Job"
fi

# dev 沒設 resources：不能輸出空的 resources 欄位
if grep -Eq 'resources: (\{\}|null)?$' <<<"$(helm template web "$chart" -f "$chart/values-dev.yaml" --set resources=null)"; then
  echo "FAIL: 沒設 resources 時不該輸出空的 resources 欄位"
  fail=1
else
  echo "ok:   沒設 resources 時不輸出空欄位"
fi

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "all checks passed"
