# 後端目錄

本領域寫給自學的初級工程師，以 Go 為主要語言，分四層帶你從第一個真實服務走到分散式系統與架構決策。

**先修**：計算機科學領域（`docs/contents/cs.md`）。機器怎麼執行程式、作業系統、網路與命令列工具都在那裡，本領域不再重講，只在需要時連回去。

| 層級 | 主題 | 讀完你能做到 |
| --- | --- | --- |
| L1 入門 | Go 語言、HTTP 服務、SQL、測試 | 獨立寫出一個有資料庫的 API |
| L2 中階 | 資料庫深入、並行、快取、佇列、容器 | 讓服務穩定上線並看得到它的狀態 |
| L3 資深 | 分散式系統、架構、效能、雲原生 | 設計能水平擴展、能容錯的系統 |
| L4 架構 | 可靠性、安全、成本、技術決策 | 為整個產品的後端方向負責 |

> 容器、CI/CD、Kubernetes、可觀測性、安全在本領域只講到「讓這個服務能上線」的程度；更深的版本在 DevOps 與資安領域，屆時會互相連結。

## L1 入門後端：用 Go 寫出第一個真實服務

L1 的目標是獨立完成一個接資料庫、有測試的 REST API。

### 1.1 Go 語言核心

- [ ] 開發環境：安裝、`go mod`、專案結構
- [ ] 型別系統：基本型別、slice 與 array、map、struct
- [ ] 函式、多回傳值、`defer`
- [ ] 指標與值語意：何時傳值、何時傳指標
- [ ] method 與 interface：Go 的組合哲學
- [ ] 錯誤處理：`error` 慣例、`errors.Is/As`、wrapping
- [ ] 套件與可見性：首字母大寫規則
- [ ] 泛型（generics）的基礎用法
- [ ] 標準庫導覽：`fmt`、`strings`、`time`、`io`、`encoding/json`

### 1.2 HTTP 服務與 API

- [ ] 請求生命週期：從 socket 到 handler
- [ ] `net/http`：Handler、ServeMux、middleware 模式
- [ ] HTTP 語意：method、狀態碼、header、冒等性初步
- [ ] REST 設計原則與資源命名
- [ ] JSON 編解碼與輸入驗證
- [ ] 統一的錯誤回應格式
- [ ] 設定管理：環境變數、設定檔
- [ ] 框架選型：標準庫 vs chi、gin、echo

### 1.3 關聯式資料庫入門

- [ ] 為什麼需要資料庫，而不是檔案
- [ ] SQL 基礎：SELECT、INSERT、UPDATE、DELETE、JOIN、GROUP BY
- [ ] 資料表設計：主鍵、外鍵、正規化初步
- [ ] 用 Go 連資料庫：`database/sql`、連線池
- [ ] 防止 SQL injection：參數化查詢
- [ ] Migration：資料庫版本控制
- [ ] ORM 與 query builder 與原生 SQL：如何選

### 1.4 認證與基礎安全

- [ ] 認證（authn）vs 授權（authz）
- [ ] 密碼雜湊：bcrypt、argon2
- [ ] Session 與 Cookie
- [ ] JWT 的基礎與限制
- [ ] 常見漏洞入門：XSS、CSRF、敏感資料外洩
- [x] CORS：原理、預檢請求與在 Go 中正確設定（CORS 主線 2/3，承 CS N.4，接資安 P2 2.3）→ [文章](../../site/src/content/docs/backend/l1/1-4-auth-basics/cors.md)

### 1.5 測試入門與程式碼品質

本節只講「用 Go 寫出測試」：語法、工具與一個可以跑的起點。該測什麼、測試替身、整合測試、測試策略在測試與品質領域（`docs/contents/testing.md`）。

- [ ] `testing` 套件與 table-driven tests
- [ ] 用介面換掉依賴：寫出第一個 fake
- [ ] `httptest` 測試 handler
- [ ] 程式碼格式與靜態分析：`gofmt`、`go vet`、linter
- [ ] 日誌入門：`log/slog`

### 1.6 小結專案

- [ ] 完成「短網址服務 v1」：單機、單一資料庫、有測試

## L2 中階：讓服務穩定、快速、看得見

L2 的目標是把一個「能跑」的服務，變成「能上線、能承受流量、出了問題找得到原因」的服務。

### 2.1 Go 並行程式設計

- [ ] goroutine 與 GMP 排程模型
- [ ] channel：緩衝與非緩衝、关閉語意
- [ ] `select` 與逐時處理
- [ ] `sync`：Mutex、RWMutex、WaitGroup、Once、Pool
- [ ] `sync/atomic` 與記憶體模型入門
- [ ] `context`：取消、逾時、傳遞請求範圍資料
- [ ] 常見並行模式：worker pool、pipeline、fan-out/fan-in、`errgroup`
- [ ] goroutine 洩漏與排查
- [ ] race detector：`go test -race`
- [ ] 優雅關機（graceful shutdown）

### 2.2 資料庫深入

- [ ] 索引原理：B+ 樹、複合索引、覆蓋索引
- [x] 主鍵選型：自增 BIGINT vs UUID vs UUIDv7／Snowflake（Postgres 為主、MySQL 對照）→ [文章](../../site/src/content/docs/backend/l2/2-2-database/primary-key-choice.md)
- [ ] 讀懂執行計畫：`EXPLAIN`
- [ ] 交易與 ACID
- [ ] 隔離級別與異常現象：dirty read、phantom read
- [ ] 鎖：行鎖、樂觀鎖、悲觀鎖、deadlock
- [ ] 慢查詢診斷與優化
- [ ] N+1 查詢問題
- [ ] 連線池調校
- [ ] 資料表設計進階：反正規化、軟刪除、分頁（offset vs cursor）
- [ ] NoSQL 全貌：鍵值、文件、列式、圖形資料庫的適用時機

### 2.3 快取

- [ ] 快取的三層：程式內、Redis、CDN
- [ ] Redis 資料結構與使用情境
- [ ] 快取模式：cache-aside、write-through、write-back
- [ ] 失效策略：TTL、LRU、主動失效
- [ ] 快取穿透、擊穿、雪崩與對策
- [ ] 快取與資料庫的一致性問題
- [ ] 分散式鎖的初步認識

### 2.4 非同步與訊息佇列

- [ ] 為什麼要非同步化
- [ ] 訊息佇列核心概念：producer、consumer、broker
- [ ] RabbitMQ 與 Kafka 的定位差異
- [ ] 投遞保證：at-most-once、at-least-once、exactly-once
- [ ] 重試、死信佇列、冒等消費
- [ ] 背壓與限流入門
- [ ] 排程任務與背景工作

### 2.5 API 設計進階

- [ ] API 版本管理與向後相容
- [ ] 幂等性：Idempotency Key
- [ ] 分頁、過濾、排序的設計
- [ ] 限流（rate limiting）算法
- [ ] OpenAPI 規格與文件
- [ ] gRPC 與 Protocol Buffers
- [ ] GraphQL 的適用與代價
- [ ] Webhook 設計

### 2.6 認證授權進階

- [ ] OAuth 2.0 與 OpenID Connect
- [ ] Access token 與 Refresh token
- [ ] RBAC 與 ABAC
- [ ] 服務間認證
- [ ] 秘密管理

### 2.7 容器與交付

- [ ] Docker：image、layer、multi-stage build
- [ ] docker compose 建立本地環境
- [ ] CI/CD：GitHub Actions 流水線
- [ ] 部署策略：滾動、藍綠、金絲雀
- [ ] 設定與環境分離（Twelve-Factor）

### 2.8 可觀測性基礎

- [ ] 日誌、指標、追蹤三支柱
- [ ] 結構化日誌與 correlation ID
- [ ] Prometheus 與 Grafana
- [ ] 健康檢查與 readiness/liveness
- [ ] 用 `pprof` 找 CPU 與記憶體瓶頸
- [ ] 告警的基本原則

### 2.9 小結專案

- [ ] 「短網址服務 v2」：加入 Redis 快取、非同步記錄點擊、Docker 化與 CI

## L3 資深：設計能水平擴展、能容錯的系統

L3 的重心從「怎麼寫」轉到「怎麼設計與取捨」，語言退居配角，但每章都用 Go 實作一個小版本來驗證理解。

### 3.1 分散式系統基礎

- [ ] 為什麼分散式很難：網路不可靠、時鐘不同步、部分失敗
- [ ] CAP 定理與 PACELC
- [ ] 一致性模型：強一致、最終一致、因果一致
- [ ] 時鐘與順序：Lamport clock、vector clock
- [ ] 複寫：主從、多主、無主
- [x] MariaDB 標準版與 Galera 叢集 → [文章](../../site/src/content/docs/backend/l3/3-1-distributed-basics/mariadb-galera.md)
- [ ] 分片（sharding）與一致性雜湊
- [ ] 共識演算法：Paxos 與 Raft
- [ ] 領導者選舉與分散式鎖
- [ ] Quorum 讀寫

### 3.2 跨服務資料一致性

- [ ] 分散式交易為什麼不好用：2PC 的問題
- [ ] Saga 模式：編排式 vs 協作式
- [ ] Outbox 模式
- [ ] 事件溯源（Event Sourcing）與 CQRS
- [ ] 冒等與去重的完整設計
- [ ] 資料複製與 CDC

### 3.3 服務架構

- [ ] 單體、模組化單體、微服務：何時該拆
- [ ] 依領域拆服務：DDD 基礎、bounded context
- [ ] 服務間通訊：同步 vs 非同步
- [ ] 服務探索與設定中心
- [ ] API Gateway 與 BFF
- [ ] 事件驅動架構
- [ ] 乾淨架構、六角架構與 Go 的實務專案分層

### 3.4 韌性設計

- [ ] 逾時、重試、指數退避與 jitter
- [ ] 斷路器、艱壁隔離
- [ ] 限流、降級、負載篡除
- [ ] 重試風暴與級聯故障
- [ ] 故障注入與混沌工程入門

### 3.5 效能調校

- [ ] 效能的度量：延遲、吞吐、p99
- [ ] 基準測試：`testing.B`、壓力測試
- [ ] Go 執行期：GC、逃逸分析、記憶體配置
- [ ] pprof、trace 進階用法
- [ ] 資料庫層的調校：讀寫分離、分區表
- [ ] 實作：將 Go 服務優化一個數量級

### 3.6 資料底層原理

- [ ] 儲存引擎：B+ 樹 vs LSM-tree
- [ ] WAL 與崩潰恢復
- [ ] MVCC
- [ ] 搜尋引擎入門：倒排索引、Elasticsearch
- [ ] 時序資料庫與列式儲存
- [ ] 資料倉儲與串流處理概觀

### 3.7 雲原生

- [ ] Kubernetes 核心：Pod、Deployment、Service、Ingress
- [ ] 設定、秘密與彈性伸縮
- [ ] 服務網格（service mesh）的取捨
- [ ] 基礎設施即程式碼：Terraform
- [ ] 雲端服務全貌：計算、儲存、網路、托管資料庫
- [ ] Serverless 的適用與限制

### 3.8 系統設計實戰

- [ ] 系統設計的思考框架：需求、估算、設計、取捨
- [ ] 容量估算（back-of-the-envelope）
- [ ] 案例：短網址、限流器、通知系統、訂單系統、搜尋建議、動態消息流
- [ ] 「短網址服務 v3」：拆服務、上 K8s、加入完整可觀測性

## L4 架構與 Staff：為方向負責

L4 談的不再是單一系統，而是如何讓多個系統、多個團隊長期運作得好，並做出站得住腳的技術決策。

### 4.1 可靠性工程

- [ ] SLI、SLO、SLA 與錯誤預算
- [ ] 值班、事故處理流程
- [ ] 無責任事後審查（postmortem）
- [ ] 容量規劃與成長預測
- [ ] 災難復原：RPO、RTO、多區域部署
- [ ] 資料備份與還原演練
- [ ] 變更管理與功能開關

### 4.2 安全與合規

- [ ] 威脅建模
- [ ] OWASP Top 10 在後端的對應
- [ ] 加密：傳輸中、靜態、金鑰管理
- [ ] 零信任與網路分區
- [ ] 供應鏈安全：相依套件掃描
- [ ] 個資與隱私法規（例如 GDPR）基礎
- [ ] 審計日誌

### 4.3 資料與平台

- [ ] 資料管線：ETL、串流、批次
- [ ] 資料湖與資料倉儲
- [ ] 內部開發者平台（Platform Engineering）
- [ ] 多租戶設計
- [ ] 大型資料遠移與零停機遷移

### 4.4 成本與商業思維

- [ ] 雲端成本結構與 FinOps
- [ ] 自建 vs 買服務
- [ ] 技術債的管理
- [ ] 把技術指標轉成商業語言

### 4.5 技術決策與團隊

- [ ] 架構決策紀錄（ADR）
- [ ] 技術選型的評估框架
- [ ] 寫設計文件與 RFC
- [ ] Code review 與工程文化
- [ ] 帶人、跨團隊溝通與影響力
- [ ] 產品化思維：從問題出發，而不是從技術出發

### 4.6 終點專題：真實世界案例讀法

- [ ] 如何讀一篇公司技術部落格
- [ ] 源碼閱讀方法：從 Go 標準庫與 etcd、Redis、Kubernetes 開始
- [ ] 持續學習的路線圖



## 全書專案：短網址服務的四次演進

同一個短網址服務貫穿全書，每一版都因為「上一版擁擠」而引入新概念，讓讀者親眼看到每個技術為什麼出現。

| 版本 | 上一版的痛點 | 新增能力 | 主要概念 | 對應章節 |
| --- | --- | --- | --- | --- |
| v1 | 從零開始 | 建立、查詢、重導向 API，資料存 SQL | HTTP、JSON、SQL、測試 | L1 |
| v2 | 讀取量變大、資料庫吃緊、點擊記錄拖慢回應 | Redis 快取、非同步記錄點擊、限流、Docker 化 | 快取模式、佇列、並行、CI/CD | L2 |
| v3 | 單一服務難以擴展與獨立部署 | 拆成寫入、讀取、分析三服務，上 K8s，完整可觀測性 | 服務拆分、冒等、韌性、分片 | L3 |
| v4 | 全球使用者、高可用要求 | 多區域部署、SLO 與告警、災難演練、成本最佳化 | 可靠性、安全、FinOps、ADR | L4 |