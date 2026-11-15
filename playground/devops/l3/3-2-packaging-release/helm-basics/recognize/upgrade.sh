#!/usr/bin/env bash
# AI 產生：把短網址服務的 chart 升級到新版映像
set -e

helm repo add podinfo https://stefanprodan.github.io/podinfo
helm repo update

helm upgrade --install web podinfo/podinfo \
  --namespace demo --create-namespace \
  --set image.tag=6.8.0 \
  --set ui.message="shortener-prod" \
  --set redis.password=s3cr3t-prod-pw

helm status web -n demo
