# YAML 120 題訓練營

從 YAML 語法到能獨立設計 Docker Compose、Kubernetes、Helm、Ansible、CI、Argo CD、Argo Workflows、Prometheus、Alertmanager 的設定。共 120 題，分 4 個級別，全程使用同一個範例應用 `shop`。

## 目錄結構

```
playground/code/yaml/
  README.md              # 本檔：學習路徑
  questions/
    L1-basics.md         # Q1–Q25
    L2-single-tool.md    # Q26–Q60
    L3-composition.md    # Q61–Q95
    L4-design.md         # Q96–Q120
  answers/
    L1-basics.md
    L2-single-tool.md
    L3-composition.md
    L4-design.md
  code/                  # 你自己寫的 YAML，一題一個檔案，例如 code/L1/Q01.yaml
  fixtures/              # 題目提供的起始檔與壞掉的 YAML（L4 除錯題用）
  check.sh               # 驗證腳本：./check.sh Q12
```

題目與解答分開存放。先自己寫，再對照 `answers/`。

## 環境

本訓練營不需要執行中的服務，只需要驗證工具。

| 工具 | 用途 | 首次使用 |
|---|---|---|
| `yamllint` | YAML 語法與風格 | L1 |
| `yq` | 用查詢斷言欄位值 | L1 |
| `docker compose config` | 驗證 Compose | L1 |
| `kubeconform` | 驗證 Kubernetes 資源 schema | L1 |
| `helm` | `lint`、`template` | L2 |
| `ansible-playbook`、`ansible-lint` | 語法檢查、最佳實務 | L2 |
| `check-jsonschema` | 用 JSON Schema 驗證 GitHub Actions、GitLab CI、Azure Pipelines | L1 |
| `kustomize` | `build` | L3 |
| `argo` CLI | `argo lint` 驗證 Workflow | L2 |
| `promtool` | `check rules`、`check config`、`test rules` | L1 |
| `amtool` | `check-config`、`config routes test` | L2 |

Argo CD 的 `Application` 與 `ApplicationSet` 用 `kubeconform` 搭配 CRD schema 驗證。

驗證工具安裝方式待補（預計寫入 `mise.toml`）。

## 情境資料集

全程使用同一個應用，換工具時不用重新理解業務。

| 元件 | 說明 | 首次使用 |
|---|---|---|
| `web` | 前端，nginx，埠 80 | L1 |
| `api` | 後端，HTTP 埠 8080，提供 `/healthz` 與 `/metrics` | L1 |
| `db` | PostgreSQL 16，埠 5432，需要持久化 | L1 |
| `redis` | 快取，埠 6379 | L2 |
| `worker` | 背景工作，消費佇列，無對外埠 | L2 |
| 環境 | `dev`、`staging`、`prod`，設定值各不相同 | L3 |

## 每題格式

1. **題目**：情境加需求。
2. **驗收標準**：用哪個工具驗證、必須通過的檢查項目（欄位值、筆數、必要欄位）。
3. **提示**（可選）：卡關時才看，分三層：方向、欄位、近乎答案。
4. **解答**（在 `answers/`）：參考 YAML、解題思路、常見錯誤。

題型分六種：

| 題型 | 說明 |
|---|---|
| 填空 | 給半成品，補齊缺的欄位 |
| 讀懂 | 給 YAML，回答它會做什麼 |
| 撰寫 | 依規格從零寫出 |
| 轉換 | 把一種格式改成另一種（Compose 轉 K8s） |
| 除錯 | 給壞掉的 YAML，找出並修正 |
| 審查 | 給能跑但有問題的 YAML，指出風險並改善 |

## 學習方法

1. 先讀完該級的「必備概念」，再開始做題。
2. 每題先自己寫，用 `./check.sh Qxx` 驗證。
3. 驗證失敗時先讀錯誤訊息，不要馬上看解答。
4. 提示由淺到深逐層打開，記下用了幾層。
5. 做完對照解答，比較寫法差異。
6. 每級最後一題是綜合題，答不出來就回頭重做該級。
7. 錯題當天寫下一句話的錯因，再做成 Anki 卡片，之後依間隔複習。
8. L2 起，每做 5 題就隨機回頭重做 1 題舊題（交錯練習）。

## 學習路徑

### L1 基礎（Q1–Q25）：能讀懂 YAML、能寫單一資源的小片段

**必備概念**：縮排、scalar、list、map、多行字串、型別推斷。

| 題號 | 主題 |
|---|---|
| Q1–Q4 | 語法：scalar、list、map、巢狀結構、註解 |
| Q5–Q6 | 多行字串：`\|` 與 `>`、`-` 與 `+` 的差異 |
| Q7–Q8 | 型別陷阱：`no`、`on`、`1.0`、`0755`、`'123'` 與 `123` |
| Q9 | Anchor 與 alias：`&`、`*`、`<<` |
| Q10 | 多文件：`---` 分隔、`kubectl` 與 `yq` 如何讀 |
| Q11–Q15 | Docker Compose 入門：`image`、`ports`、`environment`、`volumes`、`depends_on` |
| Q16–Q19 | Kubernetes Pod：`metadata`、`labels`、`containers`、`env`、`resources` |
| Q20–Q21 | GitHub Actions：`on`、`jobs`、`steps`、`uses` 與 `run` |
| Q22 | Azure Pipelines：`trigger`、`pool`、`steps`、`script` 與 `task` |
| Q23–Q24 | Prometheus 設定：`global`、`scrape_configs`、`static_configs` |
| Q25 | **綜合題**：把一份純文字需求寫成 Compose，通過 `docker compose config` |

### L2 單工具完整寫（Q26–Q60）：能獨立寫出每個工具可用的設定

**必備概念**：各工具的資源模型、必填欄位、`apiVersion` 與 `kind`、schema 驗證。

| 題號 | 主題 |
|---|---|
| Q26–Q31 | Compose：多服務、`networks`、named volume、`healthcheck`、`env_file`、`profiles` |
| Q32–Q34 | K8s 工作負載：`Deployment`、`replicas`、`selector` 與 `labels` 對應、滾動更新 |
| Q35–Q36 | K8s 網路：`Service`（ClusterIP、NodePort）、`Ingress` |
| Q37–Q38 | K8s 設定：`ConfigMap`、`Secret`，掛載為環境變數與檔案 |
| Q39 | K8s 儲存：`PersistentVolumeClaim` |
| Q40 | K8s 健康檢查與資源：`livenessProbe`、`readinessProbe`、`requests` / `limits` |
| Q41–Q43 | Helm：`Chart.yaml`、`values.yaml`、目錄結構、`helm lint` |
| Q44–Q48 | Ansible：inventory、playbook、`tasks`、`vars`、`when`、`loop`、`handlers` |
| Q49 | GitHub Actions：`matrix`、`needs`、`cache` |
| Q50 | GitLab CI：`stages`、`rules`、`artifacts` |
| Q51 | Azure Pipelines：`stages`、`jobs`、`dependsOn`、`condition`、`variables`、`strategy.matrix` |
| Q52–Q54 | Argo CD：`Application`、`source`、`destination`、`syncPolicy` |
| Q55–Q56 | Argo Workflows：`Workflow`、`templates`、`container`、`steps` |
| Q57–Q58 | Prometheus 告警規則：`groups`、`rules`、`expr`、`for`、`labels`、`annotations` |
| Q59 | Alertmanager：`route`、`receivers`、`group_by` |
| Q60 | **綜合題**：用 `shop` 一次寫出 K8s 全套（Deployment、Service、ConfigMap、Ingress） |

### L3 組合與模板（Q61–Q95）：能跨資源關聯、參數化、重用

**必備概念**：資源之間的引用關係、模板語言、覆寫與合併規則、GitOps。

| 題號 | 主題 |
|---|---|
| Q61–Q67 | Helm 模板：`{{ .Values }}`、`if`、`range`、`include` 與 `_helpers.tpl`、`toYaml`、`nindent`、`required` |
| Q68–Q71 | Kustomize：`bases`、`overlays`、`patches`、`configMapGenerator`、`images` |
| Q72–Q76 | Ansible 進階：`roles`、變數優先序、`templates`（Jinja2）、`vault`、`include_tasks` |
| Q77 | K8s `RBAC`：`Role`、`RoleBinding`、`ServiceAccount` |
| Q78 | K8s `HorizontalPodAutoscaler` |
| Q79 | K8s `NetworkPolicy` |
| Q80 | K8s `CronJob` 與 `Job` |
| Q81–Q82 | Argo CD `ApplicationSet`：list generator、git generator |
| Q83–Q84 | Argo CD：sync wave、hook、`ignoreDifferences` |
| Q85–Q86 | Argo Workflows：DAG、`dependencies`、參數與 `withItems` |
| Q87–Q88 | Argo Workflows：artifacts、`CronWorkflow`、`WorkflowTemplate` |
| Q89–Q91 | Prometheus：recording rules、命名慣例、用 `promtool test rules` 寫單元測試 |
| Q92–Q93 | Alertmanager：路由樹、`matchers`、`inhibit_rules`、`mute_time_intervals` |
| Q94 | 轉換：把 Compose 改寫成 K8s 資源 |
| Q95 | **綜合題**：把 `shop` 做成 Helm chart，`dev` 與 `prod` 用不同 values |

### L4 除錯與設計（Q96–Q120）：能找出問題、能做安全與上線等級的設計

**必備概念**：常見失誤模式、最小權限、可觀測性、GitOps 流程。

| 題號 | 主題 |
|---|---|
| Q96–Q101 | 除錯：縮排與語法錯誤、型別陷阱、Compose 起不來、K8s selector 對不上、Helm 渲染錯誤、Ansible 變數未定義 |
| Q102–Q105 | 安全加固：非 root、`readOnlyRootFilesystem`、移除 capabilities、`Secret` 不進 Git |
| Q106–Q108 | 審查：找出缺少 probe、缺少 resource limits、使用 `latest` 標籤等風險 |
| Q109–Q111 | Argo CD 設計：app-of-apps、多環境 promotion、`AppProject` 權限邊界 |
| Q112–Q113 | Argo Workflows 設計：完整 CI 流程（測試、建置、推送、通知）、失敗重試 |
| Q114–Q116 | 告警設計：SLO 與 burn rate 告警、避免 flapping、Alertmanager 分級路由與 silence |
| Q117–Q119 | 需求轉設計：從文字需求產出跨工具的完整設定 |
| Q120 | **總結題**：給一套有多處問題的 `shop` 設定，限時找出並修正所有問題 |

## 完成標準

| 級別 | 達成條件 |
|---|---|
| L1 | 不看資料也能寫出正確縮排與型別的 YAML，說得出常見型別陷阱，並寫出 Compose、Pod、CI 的小片段 |
| L2 | 能依規格寫出每個工具通過 schema 驗證的設定，說得出各資源的必填欄位 |
| L3 | 能用模板與覆寫處理多環境，能說明資源之間靠哪個欄位互相關聯，能為規則寫單元測試 |
| L4 | 能在限時內找出設定中的語法、邏輯與安全問題，能從需求獨立設計完整的設定 |

每級過關條件：該級正確率 ≥ 80%，且錯題重做後全數通過，才進入下一級。

## 進度追蹤

- [ ] 環境：`check.sh`、驗證工具安裝
- [ ] L1：Q1–Q25
- [ ] L2：Q26–Q60
- [ ] L3：Q61–Q95
- [ ] L4：Q96–Q120
