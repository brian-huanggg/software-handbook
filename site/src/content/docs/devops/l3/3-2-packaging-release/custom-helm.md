---
title: 自寫 Helm chart
description: 為自家服務寫一個可分環境部署的 chart，用 lint 與 template 驗證，並以 hook 在升級前跑 migration。
sidebar:
  order: 2
---

## 先想想

短網址服務 v3 要上 prod 了。新版加了一張資料表，所以 Pod 啟動前要先跑一次資料庫 migration。prod 有 3 個副本，而且 Deployment 每次滾動更新都會建立新的 Pod。

同事提議：在 Deployment 的 Pod 裡加一個 init container，啟動時先跑 `migrate`，成功了才啟動主程式。這樣不用另外寫什麼，看起來最省事。

這個做法在 3 個副本同時啟動時會發生什麼事？Pod 因為節點故障被重建時又會發生什麼事？如果換成「升級前先跑一次、成功了才繼續」，要用 Helm 的哪個機制表達？

<details>
<summary>提示</summary>

想想 init container 的生命週期跟誰綁在一起：是跟「一次升級」綁，還是跟「每一個 Pod」綁。再想想 Helm 能不能在送出 Deployment 之前，先執行別的東西並等它結束。

</details>

## 是什麼

**自寫 Helm chart** 是為自己的服務寫一個 chart：用 `Chart.yaml` 描述它，用 `templates/` 裡的模板產生 Kubernetes 清單，用 `values.yaml` 開出可調整的欄位，讓同一份 chart 能裝到不同環境。

上一篇「[Helm 基礎](/devops/l3/3-2-packaging-release/helm-basics/)」我們只當使用者。這一篇換成作者，要回答的是：哪些欄位該變成 values、哪些該寫死，以及怎麼在不碰叢集的情況下確認寫對了。它最接近你熟悉的前端元件加 props：模板是元件，values 是 props，`helm template` 是在畫面出來之前先看它渲染成什麼。

## 為什麼存在

不自己寫 chart，你有兩條路，各有代價。

**第一條：全用別人的 chart。** 第三方軟體可以這樣做，自家服務不行。沒有現成的 chart 知道你的服務需要先跑 migration、要引用哪個 Secret，也不知道你的 ConfigMap 變動後 Pod 需要重啟。

**第二條：手寫硬編碼的 YAML。** 這就回到上一篇的老問題：dev、staging、prod 各複製一份，過兩個月沒人知道它們差在哪。一個服務有 7 份清單、3 個環境，就是 21 份檔案要同步。

自寫 chart 把兩件事收進同一個地方。差異集中在每個環境一個小小的 values 檔；「升級前先跑 migration」這類部署邏輯寫在 chart 裡，隨版本一起走，而不是放在某個人的腦袋或 CI 腳本裡。

## 怎麼運作

我們用短網址服務寫一個 chart，完整範例在 `playground/devops/l3/3-2-packaging-release/custom-helm/shortener/`。

### chart 的結構

```text
shortener/
├── Chart.yaml
├── values.yaml            # 預設值，同時是「這個 chart 能調什麼」的文件
├── values-dev.yaml        # 每個環境只放和預設值不同的欄位
├── values-prod.yaml
└── templates/
    ├── _helpers.tpl       # 共用的命名與標籤片段，不會被渲染成清單
    ├── configmap.yaml
    ├── deployment.yaml
    ├── service.yaml
    └── migration-job.yaml # hook，下面再談
```

看 `values-dev.yaml` 與 `values-prod.yaml`：它們放在 chart 目錄裡只是方便示範，環境專屬的 values 也可以放在部署用的 Git 倉庫。

`Chart.yaml` 有兩個容易搞混的版本。`version` 是 chart 本身的版本，模板或 values 結構一改就該升；`appVersion` 是服務的版本，這裡是 1.0.0，只是給人看的資訊，除非模板主動引用它。

```yaml title="Chart.yaml"
apiVersion: v2
name: shortener
description: 短網址服務
type: application
version: 0.1.0
appVersion: "1.0.0"
```

### 模板的三個內建物件

模板用的是 Go 的 `text/template`，在 `{{ }}` 裡取值。你主要會用到三個物件：

| 物件 | 內容 | 例子 |
| --- | --- | --- |
| `.Values` | 合併後的 values | `.Values.replicaCount` |
| `.Release` | 這次安裝的資訊 | `.Release.Name`、`.Release.Namespace` |
| `.Chart` | `Chart.yaml` 的內容 | `.Chart.Name`、`.Chart.AppVersion` |

先看最簡單的 Service。它只引用了 values 與共用片段：

```yaml title="service.yaml"
apiVersion: v1
kind: Service
metadata:
  name: {{ include "shortener.fullname" . }}
  labels:
    {{- include "shortener.labels" . | nindent 4 }}
spec:
  selector:
    {{- include "shortener.selectorLabels" . | nindent 4 }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.targetPort }}
```

`include` 呼叫 `_helpers.tpl` 裡定義的片段。重複出現的名稱與標籤只寫一次，改命名規則時只改一個地方。

```yaml title="_helpers.tpl"
{{- define "shortener.fullname" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "shortener.selectorLabels" -}}
app.kubernetes.io/name: {{ .Chart.Name }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
```

看 `trunc 63`：Kubernetes 的名稱有長度上限，release 名稱加 chart 名稱可能超過，所以在這裡截斷。

`{{-` 的減號會吃掉左邊的空白與換行，`nindent 4` 則是「先換行，再把每一行縮排 4 格」。這兩個是模板裡最常出錯的地方，下面「常見坑」會再回頭。

### 條件、預設值與必填

Deployment 裡有三種常見的取值寫法：

```yaml title="deployment.yaml"
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          # ...
          {{- with .Values.resources }}
          resources:
            {{- toYaml . | nindent 12 }}
          {{- end }}
```

- `default`：`image.tag` 沒填時，改用 `appVersion`。適合「有合理預設值」的欄位。
- `with`：`resources` 有值才輸出整個區塊；沒設定時一個字都不輸出，而不是輸出一個空的 `resources:`。
- `toYaml | nindent`：把一整個巢狀結構原樣轉成 YAML，縮排到正確的位置。

ConfigMap 則用 `required`，處理「沒有合理預設值」的欄位：

```yaml title="configmap.yaml"
data:
  BASE_URL: {{ required "config.baseUrl 必須設定" .Values.config.baseUrl | quote }}
  LOG_LEVEL: {{ .Values.config.logLevel | quote }}
```

短網址產生的連結要指向哪個網域，沒有一個「對所有環境都合理」的預設值。`values.yaml` 把它留空，每個環境不給就渲染失敗，而不是悄悄用錯的值。`quote` 則保證輸出是字串：ConfigMap 的值一定要是字串，數字或布林不加引號會被 API Server 拒絕。

### 分環境的 values

預設值寫在 `values.yaml`，環境檔只列出差異：

```yaml title="values-prod.yaml"
replicaCount: 3

image:
  tag: 1.0.0

config:
  baseUrl: https://s.example.com

resources:
  requests:
    cpu: 200m
    memory: 64Mi
  limits:
    memory: 128Mi
```

合併規則沿用上一篇：chart 的 `values.yaml` 最低，`-f` 檔案其次（後面的蓋過前面的），`--set` 最高。需要注意的是，物件會逐欄位合併，但**列表會整個被取代**。所以 `migration.command` 這種列表，環境檔一旦給了就得給完整。

### 不碰叢集的驗證：lint 與 template

寫完不用先裝進叢集，先在本機做兩件事：

```bash
helm lint shortener -f shortener/values-dev.yaml
helm template web shortener -f shortener/values-prod.yaml
```

`helm lint` 檢查 chart 的結構與明顯的錯誤，例如 YAML 無法解析。它不知道你的清單在 Kubernetes 裡合不合法。`helm template` 把模板完整渲染出來，你可以直接讀結果，或用 `--show-only templates/deployment.yaml` 只看某一份。

如果想連 Kubernetes 的 schema 一起驗，需要有叢集，用 `helm install --dry-run=server`。它讓 API Server 驗證清單，但不會真的建立東西。

```text
  模板 + values ──▶ helm lint        （結構、YAML 語法）
        │
        ▼
   helm template ──▶ 讀渲染結果        （values 有沒有被套用）
        │
        ▼
   --dry-run=server ──▶ API Server 驗證 （欄位型別、必填欄位，需要叢集）
```

看這三層：越往下越接近真實叢集，但也越需要環境。大部分的錯誤在前兩層就能抓到，CI 裡只跑前兩層就很有價值。

### 用 hook 在升級前跑 migration

回到 migration。Helm 的 **hook** 讓一份清單掛在 release 生命週期的某個時間點：Helm 會在那個時間點建立它，等它完成，再繼續。

```yaml title="migration-job.yaml"
{{- if .Values.migration.enabled }}
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ include "shortener.fullname" . }}-migrate
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
    "helm.sh/hook-weight": "0"
    "helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded
spec:
  backoffLimit: 0
  # ...
```

三個 annotation 各自負責一件事：

| annotation | 作用 |
| --- | --- |
| `helm.sh/hook` | 在哪個時間點執行；`pre-install,pre-upgrade` 表示安裝與升級的清單送出之前 |
| `helm.sh/hook-weight` | 同一個時間點有多個 hook 時的順序，數字小的先跑（要寫成字串） |
| `helm.sh/hook-delete-policy` | 何時刪掉 hook 建出的資源；`before-hook-creation` 在下次建立前刪舊的，`hook-succeeded` 成功後就刪 |

升級時的順序如下：

```text
helm upgrade
   │
   ├─ 1. 渲染所有模板
   ├─ 2. 執行 pre-upgrade hook（migration Job），Helm 等它完成
   │       ├─ 成功 ─▶ 繼續
   │       └─ 失敗 ─▶ 升級中止，Deployment 不會被更新，舊版 Pod 繼續跑
   └─ 3. 送出 Deployment、Service、ConfigMap
```

看第 2 步的失敗分支：migration 失敗時，Deployment 根本沒有被送出，舊版 Pod 還在服務。這就是 hook 比 init container 安全的地方：失敗被擋在升級之前，而不是發生在新 Pod 起來之後。

`backoffLimit: 0` 讓 Job 失敗後不要自己重試，因為 migration 通常不是冪等（idempotency）的，自動重試半途失敗的 migration 比直接失敗更危險。

兩個要注意的限制。第一，hook 建立的資源不屬於 release：`helm uninstall` 不會管它，所以才需要 `hook-delete-policy`。第二，`pre-install` 的時間點比 chart 自己的物件都早，此時 ConfigMap 與 Secret 都還不存在。所以 Job 引用的資料庫 Secret 必須事先建好，不能由同一個 chart 在安裝時才建立。

### 回到一開始的問題

回到一開始的問題：init container 的生命週期綁在「每一個 Pod」，不是「每一次升級」。3 個副本同時啟動時，3 個 init container 會同時對同一個資料庫跑 migration，彼此搶鎖或重複執行；節點故障重建 Pod 時，同一個 migration 又會再跑一次。用 `pre-upgrade` hook，migration 在每次升級只跑一次，跑完 Helm 才送出 Deployment，失敗時舊版仍在服務。

## 取捨與替代方案

chart 本身有維護成本。模板是程式，會有 bug，也需要版本與測試。每多一個 `if`、`range`，就多一條要驗證的渲染路徑。

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 自寫 Helm chart | 有 release、回滾、hook；values 分環境；可發給別人用 | 模板難讀、要學 Go template；hook 行為有不少細節 | 自家服務要部署到多個環境，且有升級前後的步驟 |
| Kustomize | 沒有模板，用 patch 疊加純 YAML | 沒有 release 與 hook；大改動 patch 難寫 | 環境差異小，想維持純 YAML |
| 純 YAML＋`envsubst` | 零學習成本，一行指令 | 沒有版本記錄、沒有條件與迴圈、差異藏在腳本裡 | 一兩個環境的小專案或原型 |
| 用程式產生 YAML | 用真正的語言寫邏輯，可單元測試 | 要自己維護產生器，團隊較少人熟悉 | 清單邏輯複雜到模板難以維護 |

### 什麼時候不該用

- 只有一個環境、一兩個物件的服務。chart 的結構與 values 比它省下的重複更重。
- 模板裡已經出現多層巢狀的 `if` 與 `range`，而且沒人敢改。這時邏輯應該搬到產生 YAML 的程式，或改用 Kustomize。
- migration 需要複雜的編排（先備份、分批、可暫停）。hook 只能「跑一個 Job 並等它結束」，這種需求要用專門的遷移工具或流程。

## 常見坑與故障案例

### `indent` 與 `nindent` 用錯，YAML 壞掉

**症狀：** `helm template` 或 `helm lint` 報 `error converting YAML to JSON: yaml: line 41: did not find expected key`，行號指的是渲染後的輸出，不是模板檔。

**原因：** `toYaml` 產生的是多行文字。`indent 12` 只縮排、不換行，第一行會接在 `resources:` 後面，後面幾行又在另一個縮排位置，渲染結果的結構就壞了。`nindent 12` 會先換行再縮排。這個錯誤是用範例 chart 實際重現的。

**怎麼避免：** 凡是用 `toYaml` 輸出巢狀結構，一律寫 `{{- toYaml . | nindent N }}`，N 是它要放的欄位縮排。每改一個模板就跑 `helm lint` 與 `helm template`，錯誤在本機就看得到。

### 改了 ConfigMap，Pod 卻沒有重啟

**症狀：** 把 `logLevel` 從 `info` 改成 `debug` 後執行 `helm upgrade`，ConfigMap 已經更新，但服務的日誌層級沒變。

**原因：** 環境變數只在容器啟動時讀取。Deployment 的 Pod 範本沒有任何變化，Kubernetes 就不會觸發滾動更新，舊的 Pod 繼續用舊的值。這是示意案例，但在實務上很常見。

**怎麼避免：** 在 Pod 範本加上 annotation，內容是 ConfigMap 渲染結果的雜湊：

```yaml title="deployment.yaml"
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
```

ConfigMap 一變，雜湊就變，Pod 範本跟著變，滾動更新就會發生。

### hook Job 失敗，升級被卡住

**症狀：** migration 因為資料庫連線逾時而失敗，`helm upgrade` 回報失敗。修好之後再升級，卻報 Job 已經存在（`already exists`），或是 release 停在 `failed` 狀態。

**原因：** hook 建立的 Job 不屬於 release，Helm 不會替你清理。沒設 `hook-delete-policy` 時，失敗的 Job 會一直留著；新版 Helm 預設是 `before-hook-creation`，但舊的 chart 或明確寫了別的 policy 時仍可能碰到。另一個常見原因是 Job 引用了還不存在的 Secret，Pod 一直卡在 `CreateContainerConfigError`。這是示意案例。

**怎麼避免：** 明確寫 `before-hook-creation,hook-succeeded`：下次升級前刪掉舊的，成功後也清掉。需要查失敗原因時，`hook-succeeded` 不會刪掉失敗的 Job，可以用 `kubectl logs job/<名稱>` 看日誌。確認 Job 引用的 Secret 在第一次安裝前就已存在。

### 用 `default` 蓋過了該報錯的缺值

**症狀：** prod 上線後，短網址 API 回傳的連結全是 `http://localhost:8080/abc123`。

**原因：** 有人為了讓 `helm lint` 不報錯，把模板寫成 `{{ .Values.config.baseUrl | default "http://localhost:8080" }}`。prod 的 values 漏了這個欄位，它不但沒報錯，還安靜地套用了開發用的預設值。這是示意案例。

**怎麼避免：** 對「沒有合理預設值」的欄位用 `required`，缺了就讓渲染失敗；`default` 只留給真的有合理預設的欄位，例如 `image.tag` 預設用 `appVersion`。CI 要為每個環境的 values 都跑一次 `helm template`，缺值會在合併之前就被擋下。

## 小練習

### 動手做：補完短網址服務的 chart

目標：把半成品 chart 補齊三樣東西。起始檔案在 `playground/devops/l3/3-2-packaging-release/custom-helm/exercise/`，`starter/` 是半成品，`check.sh` 是驗收腳本，不需要叢集。

1. 在 `starter/templates/deployment.yaml` 加上 `checksum/config` annotation。
2. 同一個檔案，用 `with` 與 `toYaml | nindent` 渲染 `resources`；沒設定時不能輸出空的 `resources` 欄位。
3. 新增 `starter/templates/migration-job.yaml`：`pre-install,pre-upgrade` hook，由 `migration.enabled` 控制，Job 結束後刪除。

先執行一次 `./check.sh`，會看到幾項 `FAIL`，做完全部變成 `ok` 就完成了。

```bash
./check.sh
```

```text
ok:   helm lint
ok:   checksum/config annotation
ok:   prod 有 requests／limits 區塊
ok:   migration Job 存在
ok:   migration 是 pre-install,pre-upgrade hook
ok:   hook 結束後清掉舊 Job
ok:   migration.enabled=false 時沒有 Job
ok:   沒設 resources 時不輸出空欄位
all checks passed
```

<details>
<summary>參考解答</summary>

完整的參考答案就是 `custom-helm/shortener/`，可以用 `./check.sh ../shortener` 驗證。三處重點：

- annotation 放在 Pod 範本的 `metadata.annotations`，值是 `include (print $.Template.BasePath "/configmap.yaml") . | sha256sum`。
- `resources` 用 `{{- with .Values.resources }}` 包起來，裡面是 `{{- toYaml . | nindent 12 }}`；`values.yaml` 預設是 `resources: {}`，空物件在 `with` 裡會被視為沒有值，所以不會輸出。
- Job 整個包在 `{{- if .Values.migration.enabled }}` 裡，annotation 寫 `"helm.sh/hook": pre-install,pre-upgrade` 與 `"helm.sh/hook-delete-policy": before-hook-creation,hook-succeeded`。

</details>

### 辨識

下面是 AI 產生的 chart 片段，目標是把短網址服務部署到 prod。把它們當成 pull request 來審查，指出有什麼問題。

```yaml title="bad-chart/configmap.yaml"
# AI 產生：短網址服務的設定
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Release.Name }}-config
data:
  BASE_URL: {{ .Values.config.baseUrl | default "http://localhost:8080" | quote }}
  CACHE_TTL_SECONDS: {{ .Values.config.cacheTtlSeconds }}
```

```yaml title="bad-chart/deployment.yaml"
# AI 產生：短網址服務的 Deployment，啟動前先跑資料庫 migration
spec:
  replicas: {{ .Values.replicaCount }}
  template:
    spec:
      initContainers:
        - name: migrate
          image: "{{ .Values.image.repository }}:latest"
          command: ["/app/shortener", "migrate"]
          envFrom:
            - configMapRef:
                name: {{ .Release.Name }}-config
      containers:
        - name: shortener
          image: "{{ .Values.image.repository }}:latest"
          envFrom:
            - configMapRef:
                name: {{ .Release.Name }}-config
```

完整檔案在 `playground/devops/l3/3-2-packaging-release/custom-helm/recognize/bad-chart/`，`helm lint` 與 `helm template` 都能通過。

<details>
<summary>參考解答</summary>

`configmap.yaml`：

- `default "http://localhost:8080"`：prod 漏填 `baseUrl` 時不會報錯，短網址會指向 localhost。應該用 `required`。這對應「用 default 蓋過了該報錯的缺值」。
- `CACHE_TTL_SECONDS: {{ .Values.config.cacheTtlSeconds }}` 沒有 `quote`：渲染出來是數字 `300`，ConfigMap 的 `data` 只接受字串，`helm template` 看不出來，要到 `kubectl apply` 或 `--dry-run=server` 才被 API Server 拒絕。應該寫 `| quote`。

`deployment.yaml`：

- migration 放在 init container：3 個副本同時啟動會同時跑 3 次 migration，Pod 每次重建又會再跑一次，而且失敗時是新 Pod 起不來，不是升級被擋下。應該改成 `pre-upgrade` hook 的 Job。
- 映像標籤寫死 `latest`：每次升級拉到的內容無法預測，回滾也不保證回到同一個映像。應該用 `.Values.image.tag | default .Chart.AppVersion`，並在 prod values 固定版本。
- Pod 範本沒有 `checksum/config` annotation：之後改 ConfigMap，Pod 不會重啟，新設定不生效。
- 沒有 `resources`：prod 的 Pod 沒有資源請求與上限。應該用 `with .Values.resources` 加上，並在 prod 的 values 設定。

</details>

## 快閃卡

<details>
<summary>chart 的 <code>version</code> 與 <code>appVersion</code> 差在哪？</summary>

`version` 是 chart 本身的版本，模板或 values 結構一改就該升；`appVersion` 是被部署的服務版本，只是資訊欄位，除非模板引用它（例如 `image.tag` 的預設值）。

</details>

<details>
<summary>模板裡的 <code>.Values</code>、<code>.Release</code>、<code>.Chart</code> 各代表什麼？</summary>

`.Values` 是合併後的 values；`.Release` 是這次安裝的資訊（名稱、namespace）；`.Chart` 是 `Chart.yaml` 的內容（名稱、版本）。

</details>

<details>
<summary>為什麼設定值沒有合理預設值時，要用 <code>required</code> 而不是 <code>default</code>？</summary>

`default` 會在缺值時悄悄套用一個值，錯的環境也能渲染成功，問題要到執行時才暴露。`required` 讓缺值直接渲染失敗，錯誤出現在部署之前。

</details>

<details>
<summary>用 <code>toYaml</code> 輸出巢狀結構時，<code>indent</code> 與 <code>nindent</code> 的差別是什麼？寫錯會怎樣？</summary>

`nindent` 先換行再縮排，`indent` 只縮排。用錯時第一行會接在欄位名稱後面，渲染結果的結構壞掉，`helm lint` 與 `helm template` 會報 YAML 解析錯誤。

</details>

<details>
<summary>為什麼 migration 用 <code>pre-upgrade</code> hook 的 Job，比放在 Deployment 的 init container 好？</summary>

init container 綁在每個 Pod 上：多個副本同時啟動會重複跑，Pod 重建也會再跑，失敗時發生在新 Pod 起來之後。hook 綁在一次升級上，只跑一次，Helm 等它成功才送出 Deployment，失敗時舊版仍在服務。

</details>

<details>
<summary>你把 <code>logLevel</code> 從 <code>info</code> 改成 <code>debug</code>，<code>helm upgrade</code> 成功了，但 Pod 的日誌層級沒變。可能的原因是什麼？</summary>

Pod 範本沒有變化，Kubernetes 不會滾動更新，容器還在用啟動時讀到的環境變數。解法是在 Pod 範本加上 ConfigMap 雜湊的 `checksum/config` annotation，讓 ConfigMap 變動時 Pod 範本也跟著變。

</details>

<details>
<summary>chart 的 <code>pre-install</code> hook Job 引用了同一個 chart 建立的 Secret，第一次安裝會發生什麼事？</summary>

Job 的 Pod 卡在 `CreateContainerConfigError`。`pre-install` hook 比 chart 其他物件都早執行，此時 Secret 還不存在。Job 引用的 Secret 要事先建好，不能由同一個 chart 在安裝時才建立。

</details>

<details>
<summary>hook 的 <code>hook-delete-policy</code> 設成 <code>before-hook-creation,hook-succeeded</code>，migration Job 失敗後還看得到它嗎？為什麼要這樣設？</summary>

看得到。`hook-succeeded` 只在成功時刪除，失敗的 Job 會保留，可以用 `kubectl logs` 查原因；`before-hook-creation` 則在下次升級建立新 Job 前刪掉舊的，避免名稱衝突。

</details>

## 延伸閱讀

- [Helm 官方文件：Chart 模板指南](https://helm.sh/docs/chart_template_guide/)：內建物件、函式、流程控制與 named template 的完整說明。
- [Helm 官方文件：Chart hooks](https://helm.sh/docs/topics/charts_hooks/)：各種 hook 的時間點、權重與刪除策略。
- [Helm 官方文件：Chart 最佳實務](https://helm.sh/docs/chart_best_practices/)：命名、標籤、values 結構的建議寫法。
- [Helm 官方文件：Chart 的提示與技巧](https://helm.sh/docs/howto/charts_tips_and_tricks/)：`required`、`toYaml`、ConfigMap 變動時自動重啟 Pod 的做法。
- [helm lint](https://helm.sh/docs/helm/helm_lint/)：lint 的參數與用法。
