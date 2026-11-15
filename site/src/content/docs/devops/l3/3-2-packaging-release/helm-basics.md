---
title: Helm 基礎
description: 用 Helm 安裝現成的 chart，用 values 區分環境，並能升級、查歷史與回滾 release。
sidebar:
  order: 1
---

## 先想想

短網址服務要上 Kubernetes 了。它需要一個 Deployment、一個 Service，之後還會有 Ingress、ConfigMap 和 HPA，一共七、八份 YAML。dev 環境跑 1 個副本、用最新的映像標籤；prod 跑 3 個副本、固定版本、設了資源上限。兩個環境的 YAML 九成相同。

一個直覺的做法是複製一份目錄，改幾個欄位，兩邊各自維護。三個月後你會有 dev、staging、prod 三份，還有人在 prod 的某一份上手動 `kubectl edit` 改過東西。

這樣做到了第一次升級會出什麼問題？如果升級到一半發現新版有 bug，要怎麼確定「回到上一版」回的是同一組清單？

<details>
<summary>提示</summary>

想想三份目錄之間「哪裡不一樣」這個資訊現在存在哪裡。再想想 `kubectl apply` 套用完之後，叢集裡有沒有留下「這一批 YAML 是第幾版」的記錄。

</details>

## 是什麼

**Helm** 是 Kubernetes 的套件管理器：它把一組 Kubernetes 清單打包成可安裝、可升級、可回滾的單位，並用一份設定檔（values）決定每次安裝的差異。

它最接近你已經用過的 `apt` 或 `npm`：有倉庫、有版本、有「安裝」和「升級」指令。不同的是，Helm 安裝出來的東西跑在叢集裡，所以它還要記住「叢集裡現在裝的是哪一版、用了哪些設定」。這四個詞是整篇的基礎：

| 詞 | 意思 | 類比 |
| --- | --- | --- |
| chart | 一個套件：一組帶佔位符的清單模板，加上預設值與中繼資料 | npm 套件、`.deb` 檔 |
| values | 填進模板的設定值；每次安裝可以用自己的 values 蓋過預設值 | 設定檔、環境變數 |
| release | 一個 chart 在某個叢集、某個 namespace 裡的一次安裝，有名字與修訂版本 | 已安裝的套件實例 |
| repository | 存放 chart 的伺服器，可以是 HTTP 索引或 OCI 映像倉庫 | npm registry |

同一個 chart 可以安裝多次，每次都是獨立的 release。dev 的 `web` 和 prod 的 `web` 用的是同一份 chart，只是 values 不同。

## 為什麼存在

沒有 Helm 時，部署一個服務是 `kubectl apply -f k8s/`。這在只有一個環境時沒問題，環境一多就出現三個痛點。

**第一，清單複製後各自漂移。** dev、staging、prod 各一份目錄，差異藏在幾十個欄位裡。有人修了 prod 的探針路徑，卻忘了同步 staging，兩個月後 staging 通過的東西在 prod 行為不同。沒有一個地方明確回答「三個環境到底差在哪」。

**第二，沒有「一次安裝」這個單位。** `kubectl apply` 對每個物件各自生效，不知道這七個物件屬於同一次部署。服務改名後，舊的 ConfigMap 還留在叢集裡沒人清；要整套移除，你得自己記得有哪些物件。

**第三，沒有版本與回滾。** `kubectl apply` 不會替你記「上一批清單是什麼」。出事時你要從 Git 找回上一個 commit 的 YAML 再 apply 一次，而且前提是 Git 的狀態和叢集真的一致。

Helm 的做法是把這三件事一起處理：清單變成模板，差異集中在 values 檔；一次安裝成為一個 release，物件一起建立、一起移除；每次升級都在叢集裡留下一筆修訂記錄，可以直接回到其中任何一筆。

## 怎麼運作

### chart 的結構

一個 chart 是一個目錄，最小的結構如下：

```text
podinfo/
├── Chart.yaml        # chart 的名稱、版本、說明
├── values.yaml       # 預設的 values
└── templates/        # 帶佔位符的 Kubernetes 清單
    ├── deployment.yaml
    ├── service.yaml
    └── _helpers.tpl  # 共用的模板片段
```

看 `templates/` 裡的檔案：它們幾乎就是普通的 Kubernetes YAML，只是把會變的地方換成 `{{ .Values.replicaCount }}` 這樣的佔位符。`values.yaml` 提供每個佔位符的預設值。怎麼寫這些模板是下一篇「自寫 Helm chart」的內容，這篇只當使用者：安裝別人寫好的 chart。

### 渲染與安裝流程

執行 `helm install` 時，Helm 在你的電腦上做完所有的模板運算，再把結果送給叢集：

```text
  chart (templates + 預設 values)
        │
        ├──────── 你的 values（-f 檔案、--set）蓋過預設值
        ▼
   渲染（在本機）──▶ 純 YAML 清單
        │
        ▼
   送給 Kubernetes API Server 建立物件
        │
        ▼
   把這次安裝的記錄存成叢集裡的 Secret
   （sh.helm.release.v1.<release 名>.v<修訂號>）
```

看最後一步：Helm 不需要在叢集裡跑任何常駐程式。release 的狀態，包含當時用的 chart 與 values，被存成同一個 namespace 裡的 Secret。每次升級多一個 Secret，修訂號加一，這就是回滾能成立的原因。預設只保留最近 10 筆修訂，可用 `--history-max` 調整。

### 使用現成的 chart

我們用公開的 podinfo 當例子，它是一個很小的 Go 網頁服務，常被用來示範 Kubernetes 部署。範例檔在 `playground/devops/l3/3-2-packaging-release/helm-basics/`。

先把倉庫加進來，看 chart 提供哪些可調的值：

```bash
helm repo add podinfo https://stefanprodan.github.io/podinfo
helm search repo podinfo/podinfo
helm show values podinfo/podinfo --version 6.7.1 | head -30
```

`helm show values` 印出的就是這個 chart 的 `values.yaml`。你能調整的所有東西都在這裡，這也是讀一個陌生 chart 的第一站。

接著用我們自己的 values 檔覆蓋其中幾項：

```yaml title="values-dev.yaml"
replicaCount: 1

image:
  tag: 6.7.1

ui:
  message: "shortener-dev"

resources:
  requests:
    cpu: 50m
    memory: 32Mi
```

```yaml title="values-prod.yaml"
replicaCount: 3

image:
  tag: 6.7.1

ui:
  message: "shortener-prod"

resources:
  requests:
    cpu: 200m
    memory: 64Mi
  limits:
    memory: 128Mi
```

兩個檔案只列出和預設值不同的欄位，沒列出的欄位沿用 chart 預設值。這就是「環境差異集中在哪裡」的答案：一個小檔案，一眼看完。

安裝前先用 `helm template` 在本機渲染，不碰叢集，確認會產生什麼：

```bash
helm template web podinfo/podinfo --version 6.7.1 -f values-prod.yaml | grep -n "replicas\|image:"
```

```text
38:  replicas: 3
57:          image: "ghcr.io/stefanprodan/podinfo:6.7.1"
```

確認沒問題後安裝，`web` 是 release 名稱：

```bash
helm install web podinfo/podinfo --version 6.7.1 -f values-dev.yaml -n demo --create-namespace
```

values 有兩種給法，也有優先順序。越後面的越優先：chart 的預設 `values.yaml`，然後是 `-f` 指定的檔案（多個 `-f` 時後面的蓋過前面的），最後是 `--set`。所以 `--set replicaCount=5` 會蓋過 `-f values-prod.yaml` 裡的 3。範例資料夾的 `check.sh` 就用 `helm template` 驗證了這個順序。

:::tip
`--version` 一律固定。不寫的話 Helm 會裝最新版的 chart，同一條指令今天和下個月可能裝出不同的東西。
:::

### release 的生命週期

安裝完之後，這幾個指令涵蓋日常操作：

```bash
helm list -n demo                       # 這個 namespace 裡有哪些 release
helm status web -n demo                 # 目前狀態與說明
helm upgrade web podinfo/podinfo --version 6.7.1 -f values-prod.yaml -n demo
helm history web -n demo                # 每次修訂
helm rollback web 1 -n demo             # 回到修訂 1
helm uninstall web -n demo              # 移除整個 release
```

`helm history` 的輸出長這樣（示意）：

```text
REVISION  STATUS      CHART           DESCRIPTION
1         superseded  podinfo-6.7.1   Install complete
2         deployed    podinfo-6.7.1   Upgrade complete
```

回滾有一個容易誤會的地方：`helm rollback web 1` 不是「刪掉修訂 2」，而是用修訂 1 的內容建立一個新的修訂 3。歷史只會往前長，你永遠看得到「曾經升級過、又回滾過」。

升級時可以加兩個保護旗標：`--wait` 讓指令等到所有 Pod 都就緒才算成功；`--atomic` 在升級失敗時自動回滾到上一版（新版 Helm 另有較新的旗標名稱，以 `helm upgrade --help` 為準）。沒有這兩個旗標，Helm 只確認「清單送出去了」，不確認「服務起來了」。

### 回到一開始的問題

回到一開始的問題：複製三份目錄之後，環境差異散落在三份 YAML 裡，沒有任何地方記錄「差在哪」，叢集也不記得「這批 YAML 是第幾版」，所以回滾只能靠 Git 與記憶，而且無法保證回的是同一組清單。用 Helm 之後，差異集中在 `values-dev.yaml` 和 `values-prod.yaml` 兩個小檔案；每次升級都在叢集裡留下一筆含 chart 與 values 的修訂，`helm rollback` 回到的是那一筆的完整內容。

## 取捨與替代方案

Helm 不是免費的。它多了一層模板語言：你看到的不是最終的 YAML，要用 `helm template` 才看得到。chart 的 values 結構由 chart 作者決定，不同 chart 的欄位命名不一致，每裝一個新 chart 都得先讀它的 `values.yaml`。

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 純 YAML＋`kubectl apply` | 最直接，看到的就是送出的 | 多環境複製漂移，沒有版本與整組移除 | 單一環境、少量物件、快速實驗 |
| Helm | 打包、values 分環境、修訂與回滾、龐大的現成 chart 生態 | 模板難讀、values 結構各家不同、release 狀態存在叢集裡 | 安裝第三方軟體；自家服務要發給多個環境 |
| Kustomize | 沒有模板，用疊加（patch）改 YAML，內建於 `kubectl` | 沒有 release 與回滾概念，大改動時 patch 難寫 | 自家清單的環境差異不大，想保持純 YAML |
| Operator | 能管理有狀態軟體的整個生命週期（備份、故障轉移） | 要多裝控制器與 CRD，自己寫的成本高 | 資料庫、訊息佇列等需要營運邏輯的軟體 |

多數團隊兩者並用：第三方軟體用 Helm 裝，自家服務視團隊偏好用 Helm 或 Kustomize。

### 什麼時候不該用

- 只有一個環境、幾個物件的服務。多出來的模板層，比它省下的重複更貴。
- 清單裡需要大量條件判斷與迴圈。模板會變成難以測試的程式，這時應該考慮把邏輯移到產生 YAML 的程式裡，或改用 Kustomize。
- 軟體需要複雜的升級流程（先備份、再遷移、再切換）。Helm 只會套用清單；這類需求交給 Operator 比較合適。

## 常見坑與故障案例

### `--set` 的設定在下次升級消失

**症狀：** 上線時用 `--set replicaCount=5` 擴到 5 個副本。兩週後別人升級映像版本，副本數回到 chart 預設的 1，服務在尖峰時被打爆。

**原因：** `helm upgrade` 預設不沿用上次的 `--set`。沒有再次給出的值，會回到 chart 的預設值。這是示意案例，但在實務上很常見。

**怎麼避免：** 所有覆寫值都寫進 values 檔並放進 Git，升級時每次都用 `-f` 完整帶上。避免用 `--reuse-values`，它會連 chart 新版加入的預設值一起凍住，造成另一種難查的不一致。

### 機密寫進 values 與修訂記錄

**症狀：** 為了方便，把資料庫密碼用 `--set db.password=...` 傳進去。幾天後有人在 CI 日誌看到整條指令，或用 `kubectl get secret` 讀取 release Secret，發現密碼就在裡面。

**原因：** release Secret 裡存了當時完整的 values，只是做過編碼壓縮，並未加密。有權限讀 namespace Secret 的人就能解開。命令列也會留在 shell 歷史與 CI 日誌。

**怎麼避免：** 機密不要經過 Helm values。讓 chart 引用事先建立好的 Secret，或交給外部秘密工具（見本領域的集中式秘密管理與 HashiCorp Vault 章節）。

### 升級卡在 `pending-upgrade`

**症狀：** 升級指令被中斷（網路斷線、CI 逾時）。再次執行時出現 `another operation (install/upgrade/rollback) is in progress`，之後的升級都失敗。

**原因：** Helm 在開始操作時把修訂標成 `pending-upgrade`，操作正常結束才改狀態。中途被殺掉，標記就留在那裡，Helm 以為還有人在操作。這是示意案例。

**怎麼避免：** 先用 `helm history` 確認狀態，再用 `helm rollback` 回到最後一個 `deployed` 的修訂來解開。升級時加上 `--atomic` 與合理的 `--timeout`，讓失敗走回滾，而不是卡在中間。CI 裡避免對同一個 release 同時跑兩個升級。

### CRD 不會跟著升級

**症狀：** chart 升級後，新版的自訂資源欄位在叢集裡不存在，建立資源時被 API Server 拒絕。

**原因：** chart 的 `crds/` 目錄裡的 CRD 只在 `helm install` 時安裝一次，之後 `upgrade` 與 `uninstall` 都不會動它。這是 Helm 刻意的設計，避免誤刪 CRD 連帶清掉叢集中所有該類型的資源。

**怎麼避免：** 升級有 CRD 的 chart 前讀它的升級說明；常見做法是在升級前用 `kubectl apply` 手動更新 CRD，或把 CRD 拆成獨立的 chart 管理。

### 設計上的教訓：Tiller 被移除

Helm 2 需要在叢集裡跑一個叫 Tiller 的伺服器元件，它預設擁有很高的權限，等於讓任何能連到它的人間接取得叢集管理權。Helm 3 移除了 Tiller，改成由 Helm 用戶端直接使用你自己的 kubeconfig 權限，release 狀態改存在 Secret 裡。細節見官方的 [Helm 3 與 Helm 2 的差異](https://helm.sh/docs/faq/changes_since_helm2/)。這給我們的提醒是：Helm 能做的事，就是你的 kubeconfig 能做的事，CI 用的憑證要依最小權限給。

## 小練習

### 動手做：把一個 chart 裝進、升級、回滾

目標：用 podinfo chart 完成一輪 release 的生命週期。起始檔案在 `playground/devops/l3/3-2-packaging-release/helm-basics/`，裡面有 `values-dev.yaml`、`values-prod.yaml` 和 `check.sh`。

1. 先執行 `./check.sh`，它不需要叢集，會用 `helm template` 檢查 dev 渲染出 1 個副本、prod 渲染出 3 個副本，而 `--set` 能蓋過 `-f`。四項都顯示 `ok` 就代表你的環境與 Helm 可用。
2. 有任何本機 Kubernetes 的話（Docker Desktop、kind、minikube 皆可），用 `values-dev.yaml` 安裝 release `web`，再用 `values-prod.yaml` 升級。
3. 執行 `helm history web -n demo`，應該看到 2 筆修訂。接著 `helm rollback web 1 -n demo`，確認歷史變成 3 筆，而 `kubectl get deploy -n demo` 的副本數回到 1。

```bash
./check.sh
```

```text
ok:   dev 一個副本
ok:   prod 三個副本
ok:   prod 設了記憶體上限
ok:   --set 蓋過 -f
all checks passed
```

<details>
<summary>參考解答</summary>

```bash
helm install web podinfo/podinfo --version 6.7.1 -f values-dev.yaml -n demo --create-namespace
helm upgrade web podinfo/podinfo --version 6.7.1 -f values-prod.yaml -n demo
helm history web -n demo
helm rollback web 1 -n demo
helm history web -n demo
helm uninstall web -n demo
```

回滾後歷史有 3 筆：修訂 3 的內容等於修訂 1。如果副本數沒有回到 1，檢查是不是沒等 Pod 縮完就去查。

</details>

### 辨識

下面是 AI 產生的升級指令，和一份 prod 用的 values 覆寫。把它們當成 pull request 來審查，指出有什麼問題。

```bash title="upgrade.sh"
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
```

```yaml title="prod-values.yaml"
# AI 產生：prod 的 values 覆寫
replicaCount: 3
image:
  tag: latest
resources: {}
```

程式碼在 `playground/devops/l3/3-2-packaging-release/helm-basics/recognize/`。

<details>
<summary>參考解答</summary>

`upgrade.sh` 的問題：

- 沒有 `--version`：chart 版本不固定，同一條指令在不同時間會裝出不同的 chart。改成 `--version 6.7.1`。
- 覆寫值全用 `--set`，沒有進 Git 的 values 檔：下次有人升級忘了帶這些旗標，設定就回到預設值。改成 `-f values-prod.yaml`。
- `--set redis.password=s3cr3t-prod-pw`：密碼會留在 shell 歷史、CI 日誌，也會被存進 release Secret。改成引用事先建立的 Secret。
- 沒有 `--wait` 或 `--atomic`，也沒有 `--timeout`：失敗時不會自動回滾，中斷還可能留下 `pending-upgrade`。
- `set -e` 之後 `helm status` 只印狀態，不能代表升級成功。應該靠 `--wait` 讓指令本身在失敗時回傳非零。

`prod-values.yaml` 的問題：

- `image.tag: latest`：映像版本不固定，升級與回滾都無法預測，回滾回去的修訂可能拉到和當初不同的映像。
- `resources: {}`：prod 的 Pod 沒有資源請求與上限，會被排程器隨意安排，也可能吃光節點記憶體。參考 `values-prod.yaml` 設定 requests 與 limits。

</details>

## 快閃卡

<details>
<summary>chart、values、release 三個詞各指什麼？</summary>

chart 是可安裝的套件（模板加預設值）；values 是填進模板的設定值；release 是一個 chart 在某個叢集與 namespace 的一次安裝，有名字與修訂號。同一個 chart 可以裝成多個 release。

</details>

<details>
<summary>用 <code>kubectl apply -f</code> 管理三個環境的複製 YAML，為什麼比 Helm 更容易出問題？</summary>

環境差異散落在各份複製的 YAML 中，沒有單一位置可看出差在哪，容易漂移；而且 `kubectl apply` 不記錄「這批清單是第幾版」，也不知道哪些物件屬於同一次部署，所以沒有整組移除和回滾。

</details>

<details>
<summary>執行 <code>helm install</code> 時，模板在哪裡被渲染？release 的狀態存在哪裡？</summary>

模板在執行 `helm` 的本機渲染成純 YAML，再送給 API Server。release 的狀態存成同一個 namespace 裡的 Secret，名稱像 `sh.helm.release.v1.web.v2`，不需要叢集裡的常駐程式。

</details>

<details>
<summary>chart 預設 <code>replicaCount: 1</code>，你執行 <code>helm install web ... -f values-prod.yaml --set replicaCount=5</code>，而 values-prod.yaml 寫 3，最後會有幾個副本？</summary>

5 個。優先順序由低到高是 chart 預設值、`-f` 檔案（後面的蓋過前面的）、`--set`。

</details>

<details>
<summary><code>helm rollback web 1</code> 之後，<code>helm history</code> 會變短嗎？</summary>

不會。回滾是用修訂 1 的內容建立一個新的修訂，歷史只會增加。例如原本有修訂 1 和 2，回滾後會多出修訂 3，內容等於修訂 1。

</details>

<details>
<summary>上週用 <code>--set replicaCount=5</code> 擴容，今天同事只執行 <code>helm upgrade web ... --set image.tag=6.8.0</code>，副本數會是多少？為什麼？</summary>

回到 chart 預設值。`helm upgrade` 預設不沿用上次的 `--set`，沒有再給的值就用預設值。要把覆寫值寫進 Git 裡的 values 檔，每次升級都用 `-f` 帶上。

</details>

<details>
<summary>為什麼不該用 <code>--set db.password=xxx</code> 傳機密給 Helm？</summary>

指令會留在 shell 歷史與 CI 日誌；而且 release Secret 會存下完整的 values，只有編碼壓縮、沒有加密，有權限讀 Secret 的人就能取得密碼。機密應該引用事先建立的 Secret 或外部秘密工具。

</details>

<details>
<summary>chart 的 <code>crds/</code> 目錄裡的 CRD，執行 <code>helm upgrade</code> 時會更新嗎？為什麼這樣設計？</summary>

不會。CRD 只在 `helm install` 時安裝一次，`upgrade` 與 `uninstall` 都不碰它。因為刪除 CRD 會連帶清掉叢集裡該類型的所有資源，Helm 選擇保守。

</details>

## 延伸閱讀

- [Helm 官方文件：使用 Helm](https://helm.sh/docs/intro/using_helm/)：release、values 與升級回滾指令的完整說明。
- [Helm 官方文件：Values 檔案](https://helm.sh/docs/chart_template_guide/values_files/)：values 的優先順序與覆寫規則。
- [Helm 官方文件：Chart 的最佳實務](https://helm.sh/docs/chart_best_practices/)：看懂別人寫的 chart，也為下一篇自寫 chart 預習。
- [Helm 官方文件：Helm 3 與 Helm 2 的差異](https://helm.sh/docs/faq/changes_since_helm2/)：Tiller 為什麼被移除，以及 release 狀態如何儲存。
- [podinfo](https://github.com/stefanprodan/podinfo)：本篇範例用的 chart 與服務原始碼。
