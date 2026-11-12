# DevOps 目錄

本領域寫給已經會寫一個後端服務、想把它自動建置、部署到正式環境並讓它長期穩定運作的人。
主要工具是 Linux、Git、容器、Terraform、Ansible、Kubernetes 與 Prometheus 一家；範例以 Bash、YAML、HCL 為主，不是 Go。

**先修**：計算機科學領域的作業系統、網路、命令列與 Git 章節（`docs/contents/cs.md`）。

**貫穿專案**：把後端領域的短網址服務當作「要營運的系統」，從手動架一台機器開始，一路演進到用 GitOps 管理整座叢集。
每一層都讓上一層的痛點推著你採用新工具，而不是為了用工具而用工具。

| 層級 | 主題 | 讀完你能做到 |
| --- | --- | --- |
| L0 系統管理 | Linux 主機、服務、網路、腳本 | 讓一台 Linux 主機穩定地跑著服務 |
| L1 交付流水線 | Git 工作流、CI、容器映像、倉庫 | 讓程式自動變成可部署的映像 |
| L2 基礎設施即程式碼 | Terraform、Ansible、環境、秘密 | 用程式碼重建一整套環境 |
| L3 叢集營運 | Kubernetes、Helm、GitOps、可觀測性 | 在叢集上部署並營運服務 |
| L4 SRE 與平台 | SLO、事故、容量、FinOps、平台工程 | 為可靠性與成本負責 |

寫目錄時沿用本檔格式：`##` 為層級、`### <節號> <名稱>` 為節、`- [ ]` 為文章。
每新增一節，同時在 `site/src/sidebar/devops.ts` 加上該節的標籤與目錄名稱。

## L0 系統管理：讓一台主機活著

從單一台 Linux 主機開始，把服務手動架起來、讓它開機自啟、會輪替日誌、出事看得到。先感受痛，後面的自動化才有意義。

### 0.1 Linux 主機基礎

- [ ] 套件管理、使用者與權限、檔案系統佈局
- [ ] systemd：unit、service、把服務設成開機自啟
- [ ] journald 與日誌輪替
- [ ] 網路設定、防火牆入門

### 0.2 Shell 自動化

- [ ] 可重複執行的 Shell 腳本：`set -euo pipefail`、冪等
- [ ] cron 與 systemd timer 排程
- [ ] 把短網址服務以 systemd 服務手動部署到一台主機

### 0.3 除錯與觀察一台主機

- [ ] 看 CPU、記憶體、磁碟、網路：`top`、`ss`、`df`、`iostat`
- [ ] 追一個行程、追一條連線
- [ ] 常見「服務起不來」的排查順序

## L1 交付流水線：讓程式自動變成映像

把「手動 scp、手動重啟」換成一條流水線：推上程式，CI 自動測試、建置映像、推進倉庫。

### 1.1 Git 工作流

- [ ] 分支策略：trunk-based 與 feature branch 的取捨
- [ ] PR 與 code review 流程
- [ ] 語意化版本與標籤、變更紀錄

### 1.2 容器映像

- [ ] 從 Dockerfile 到映像：layer、快取、multi-stage build
- [ ] 把映像做小、做安全：非 root、最小基底
- [x] Distroless 映像：去掉 shell 與套件管理器以縮小攻擊面，及其除錯取捨 → [文章](../../site/src/content/docs/devops/l1/1-2-container-image/distroless-image.md)
- [ ] Golden Image：組織核准的 base image、固定 digest、掃描與簽章，以及如何強制只能從它開始
- [ ] 映像倉庫：推送、拉取、標籤策略

### 1.3 持續整合（CI）

- [ ] CI 的核心概念：觸發、工作、產物、快取
- [ ] 一條流水線：lint、測試、建置、掃描、推映像
- [ ] 把機密放進 CI 的正確方式（不寫進 repo）

### 1.4 小結

- [ ] 為短網址服務建一條 CI：提交即自動測試並產出映像推進倉庫

## L2 基礎設施即程式碼：用程式碼重建環境

不再手點主機。用 Terraform 描述要哪些資源，用 Ansible 描述機器要長成什麼樣，讓整套環境可以砍掉重建。

### 2.1 Terraform 基礎

- [ ] 宣告式思維、provider、resource、state
- [ ] 變數、輸出、模組化
- [ ] plan 與 apply 的工作流、state 放哪裡
- [ ] 用 Terraform 開出跑短網址服務所需的機器與網路

### 2.2 組態管理（Ansible）

- [ ] playbook、inventory、role
- [ ] 冪等的組態：把 L0 手動做的事寫成 Ansible
- [ ] 機密：vault 與外部秘密來源

### 2.3 環境與秘密

- [ ] dev／staging／prod 的分離
- [ ] 秘密管理：集中式秘密庫的概念與取捨
- [ ] 設定與環境分離（Twelve-Factor 回顧）

## L3 叢集營運：在 Kubernetes 上跑服務

單機不夠了。把服務搬上 Kubernetes，用宣告式的方式部署、升級、回滾，並讓整個流程由 Git 驅動。

### 3.1 Kubernetes 核心

- [ ] Pod、Deployment、Service、Ingress 各自解決什麼
- [ ] ConfigMap、Secret、資源限制與請求
- [ ] 探針：liveness、readiness、startup
- [ ] 把短網址服務部署上一座單節點叢集

### 3.2 套件化與發布

- [ ] Helm：chart、values、release
- [ ] 升級、回滾、版本管理
- [ ] 部署策略：滾動、藍綠、金絲雀

### 3.3 GitOps

- [ ] 宣告式交付：期望狀態放在 Git，由控制器收斂
- [ ] pull-based 部署的優點與邊界
- [ ] 把映像更新自動帶進叢集

### 3.4 可觀測性

- [ ] 三支柱回顧：日誌、指標、追蹤
- [ ] Prometheus 蒐集指標、Grafana 看板
- [ ] 集中式日誌（例如 Loki）
- [ ] 分散式追蹤入門
- [ ] 告警：從指標到可行動的通知

## L4 SRE 與平台：為可靠性與成本負責

工具都齊了，問題變成「怎麼讓它長期好好地跑、花得合理、出事能復原」。這一層程式碼變少，判斷變多。

### 4.1 可靠性工程

- [ ] SLI、SLO、SLA 與錯誤預算
- [ ] 值班與事故處理流程
- [ ] 無責任事後審查（postmortem）

### 4.2 容量與韌性

- [ ] 容量規劃與自動擴縮
- [ ] 混沌工程入門：主動注入故障
- [ ] 災難復原：RPO、RTO、備份與還原演練

### 4.3 成本與平台

- [ ] 雲端成本結構與 FinOps 入門
- [ ] 內部開發者平台（Platform Engineering）：把上面所有工具包成自助服務
- [ ] 自建 vs 買服務的取捨

### 4.4 收尾

- [ ] 把短網址服務的整套交付與營運，寫成一份能交接的 runbook 與平台說明
