# 自寫 Helm chart 練習

`starter/` 是短網址服務 chart 的半成品，缺三樣東西，請補齊：

1. `templates/deployment.yaml`：加上 `checksum/config` annotation，讓 ConfigMap 變動時 Pod 重啟。
2. 同一個檔案：用 `with` 與 `toYaml` 渲染 `resources`，沒設定時不輸出。
3. 新增 `templates/migration-job.yaml`：`pre-install,pre-upgrade` hook，由 `migration.enabled` 控制，hook 結束後刪除舊的 Job。

```bash
./check.sh            # 檢查 starter
./check.sh ../shortener   # 檢查完整參考答案
```

需要 Helm 3，不需要叢集。
