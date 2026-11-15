---
title: 後端目錄
description: 後端領域所有章節一覽，包含尚未寫好的章節。
sidebar:
  order: 0
---

<!-- 由 scripts/gen-toc.mjs 從 docs/contents/ 產生，請勿手動修改；改目錄請改 docs/contents/ 後重跑 `pnpm toc`。 -->

下列為後端領域規劃的所有章節。左側欄只列出已寫好的章節，其餘陸續補上。

## L1 入門後端：用 Go 寫出第一個真實服務

L1 的目標是獨立完成一個接資料庫、有測試的 REST API。

### 1.1 Go 語言核心

| 標題 | 說明 |
| --- | --- |
| 開發環境 | 安裝、`go mod`、專案結構 |
| 型別系統 | 基本型別、slice 與 array、map、struct |
| 函式、多回傳值、`defer` |  |
| 指標與值語意 | 何時傳值、何時傳指標 |
| method 與 interface | Go 的組合哲學 |
| 錯誤處理 | `error` 慣例、`errors.Is/As`、wrapping |
| 套件與可見性 | 首字母大寫規則 |
| 泛型基礎用法 | generics |
| 標準庫導覽 | `fmt`、`strings`、`time`、`io`、`encoding/json` |

### 1.2 HTTP 服務與 API

| 標題 | 說明 |
| --- | --- |
| [請求生命週期](/backend/l1/1-2-http-api/request-lifecycle/) | 說出一個 HTTP 請求從 TCP 連線到 handler 的每個階段，並為各階段設好逾時與大小限制。 |
| `net/http` | Handler、ServeMux、middleware 模式 |
| HTTP 語意 | method、狀態碼、header、冒等性初步 |
| REST 設計原則與資源命名 |  |
| JSON 編解碼與輸入驗證 |  |
| 統一的錯誤回應格式 |  |
| 設定管理 | 環境變數、設定檔 |
| 框架選型 | 標準庫 vs chi、gin、echo |

### 1.3 關聯式資料庫入門

| 標題 | 說明 |
| --- | --- |
| 為什麼需要資料庫，而不是檔案 |  |
| SQL 基礎 | SELECT、INSERT、UPDATE、DELETE、JOIN、GROUP BY |
| 資料表設計 | 主鍵、外鍵、正規化初步 |
| 用 Go 連資料庫 | `database/sql`、連線池 |
| 防止 SQL injection | 參數化查詢 |
| Migration | 資料庫版本控制 |
| golang-migrate | 版本檔命名、up／down、啟動時套用 vs CI 套用、dirty 狀態處理 |
| Flyway migration | Spring Boot 整合、版本式與 repeatable、checksum 失敗處理、undo 的限制 |
| ORM 與 SQL 選型 | ORM、query builder 與原生 SQL：如何選 |

### 1.4 認證與基礎安全

| 標題 | 說明 |
| --- | --- |
| [認證與授權的差異](/backend/l1/1-4-auth-basics/authn-vs-authz/) | 能分清「你是誰」與「你能做什麼」，並在 Go 服務中把兩者拆成各自的一層，擋下 IDOR。 |
| [密碼雜湊](/backend/l1/1-4-auth-basics/password-hashing/) | 知道為什麼密碼不能用 SHA-256 存，並能用 argon2id 加 salt 與 pepper 寫出可驗證、可升級的密碼儲存。 |
| [Session 與 Cookie](/backend/l1/1-4-auth-basics/session-cookie/) | 能用 Go 實作伺服器端 session 登入，並設定正確的 Cookie 屬性、處理登出與 session fixation。 |
| [JWT 的基礎與限制](/backend/l1/1-4-auth-basics/jwt-basics/) | 能看懂 JWT 的三段結構，用 Go 簽發與驗證 HS256 token，並說出它不能加密、難撤銷的限制。 |
| [Session vs Token 選型](/backend/l1/1-4-auth-basics/session-vs-token/) | 能說出 session 與 token 把狀態放在哪裡、撤銷得了什麼，並依情境選型。 |
| [常見漏洞入門](/backend/l1/1-4-auth-basics/common-vulnerabilities/) | 認得三種最常見的 Web 漏洞，知道後端各要做哪一道防禦，並知道深入內容在資安領域的哪裡。 |
| [CORS 與預檢請求](/backend/l1/1-4-auth-basics/cors/) | 看懂 CORS 錯誤從哪來，並用白名單在 Go 裡安全地開放跨源讀取。 |

### 1.5 測試入門與程式碼品質

本節只講「用 Go 寫出測試」：語法、工具與一個可以跑的起點。該測什麼、測試替身、整合測試、測試策略在測試與品質領域。

| 標題 | 說明 |
| --- | --- |
| `testing` 套件 | table-driven tests |
| 用介面換掉依賴 | 寫出第一個 fake |
| `httptest` | 測試 handler |
| 程式碼格式與靜態分析 | `gofmt`、`go vet`、linter |
| 日誌入門 | `log/slog` |

### 1.6 小結專案

| 標題 | 說明 |
| --- | --- |
| 短網址服務 v1 | 單機、單一資料庫、有測試 |

## L2 中階：讓服務穩定、快速、看得見

L2 的目標是把一個「能跑」的服務，變成「能上線、能承受流量、出了問題找得到原因」的服務。

### 2.1 Go 並行程式設計

| 標題 | 說明 |
| --- | --- |
| goroutine 與 GMP 排程模型 |  |
| channel | 緩衝與非緩衝、关閉語意 |
| `select` 與逐時處理 |  |
| `sync` | Mutex、RWMutex、WaitGroup、Once、Pool |
| `sync/atomic` | 記憶體模型入門 |
| `context` | 取消、逾時、傳遞請求範圍資料 |
| 常見並行模式 | worker pool、pipeline、fan-out/fan-in、`errgroup` |
| goroutine 洩漏與排查 |  |
| race detector | `go test -race` |
| 優雅關機 | graceful shutdown |

### 2.2 資料庫深入

| 標題 | 說明 |
| --- | --- |
| [正規化與反正規化](/backend/l2/2-2-database/normalization/) | 用函數相依判斷資料表該不該拆，把短網址服務的 links 表一路拆到 3NF 與 BCNF，並學會何時該刻意複製資料。 |
| 資料型別選擇 | 整數、`NUMERIC` 與金額、`timestamptz` 與時區、文字長度、布林 |
| [以資料庫時間作為唯一時鐘](/backend/l2/2-2-database/db-clock/) | 用 now() 與 timestamptz 判斷過期，讓多節點得到同一個答案，並避開時鐘漂移的陷阱。 |
| 約束 | `NOT NULL`、`UNIQUE`、`CHECK`、外鍵與 `ON DELETE` 策略 |
| 讓資料庫守住不變量 | 複合 `UNIQUE`、`CHECK`、原子 upsert，用約束取代 check-then-insert 的競態 |
| `NULL` 的語意 | 三值邏輯、`NOT IN` 的陷阱、何時該用預設值 |
| 關聯設計 | 一對多、多對多與關聯表、自我參照、多型關聯為什麼不好 |
| 資料表設計反模式 | EAV、逗號分隔欄位、JSON 濫用、過度索引 |
| 命名慣例與通用欄位 | `created_at`、`updated_at`、稽核欄位 |
| Schema 演進 | 向後相容的 migration、expand-contract、零停機改欄位 |
| 索引原理 | B+ 樹、複合索引、覆蓋索引 |
| [主鍵選型](/backend/l2/2-2-database/primary-key-choice/) | 用 Postgres 與 MySQL 實測三種主鍵的插入、索引大小與分散式代價，學會挑主鍵，並把「對外 ID」與「內部主鍵」分開。 |
| 讀懂執行計畫 | `EXPLAIN` |
| 交易與 ACID |  |
| 隔離級別與異常現象 | dirty read、phantom read |
| [資料庫鎖](/backend/l2/2-2-database/database-locks/) | 在 PostgreSQL 實測行鎖、悲觀鎖、樂觀鎖與 deadlock，學會依衝突頻率選擇鎖的策略，並用固定加鎖順序避開死結。 |
| 版本欄位與條件更新 | 樂觀鎖的 SQL 實作與衝突處理（`UPDATE ... WHERE version = ?`） |
| [PostgreSQL vs MySQL](/backend/l2/2-2-database/postgres-vs-mysql/) | 用同一張短網址資料表在兩個資料庫實測 MVCC、隔離級別與連線模型的差異，學會依工作負載選型，並避開換庫時最常踩的坑。 |
| 慢查詢診斷與優化 |  |
| N+1 查詢問題 |  |
| 連線池調校 |  |
| 資料表設計進階 | 反正規化、軟刪除、分頁（offset vs cursor） |
| NoSQL 全貌 | 鍵值、文件、列式、圖形資料庫的適用時機 |

### 2.3 快取

| 標題 | 說明 |
| --- | --- |
| 快取的三層 | 程式內、Redis、CDN |
| Redis 資料結構與使用情境 |  |
| 快取模式 | cache-aside、write-through、write-back |
| 失效策略 | TTL、LRU、主動失效 |
| 快取穿透、擊穿、雪崩與對策 |  |
| 快取與資料庫的一致性問題 |  |
| 分散式鎖的初步認識 |  |

### 2.4 非同步與訊息佇列

| 標題 | 說明 |
| --- | --- |
| 為什麼要非同步化 |  |
| 訊息佇列核心概念 | producer、consumer、broker |
| RabbitMQ 與 Kafka | 定位差異 |
| 投遞保證 | at-most-once、at-least-once、exactly-once |
| 重試、死信佇列、冒等消費 |  |
| 背壓與限流入門 |  |
| 排程任務與背景工作 |  |
| 資料庫驅動的待辦表 | `pending_*` 表、多副本下領取任務、租約與重複執行的冪等 |

### 2.5 API 設計進階

| 標題 | 說明 |
| --- | --- |
| API 版本管理與向後相容 |  |
| 幂等性 | Idempotency Key |
| 條件請求與樂觀並發控制 | ETag、`If-Match`、412 與 428 |
| 錯誤判斷的固定順序 | 401、CSRF、404、403、412／428、422 為什麼不能亂，以及統一錯誤格式（RFC 9457） |
| 分頁、過濾、排序的設計 |  |
| 分頁上限與排序穩定性 | `LIMIT` 上限、tie-breaker、排序欄位白名單與參數化 |
| 限流算法 | rate limiting |
| OpenAPI 規格與文件 |  |
| spec-first 與程式碼生成 | 用 oapi-codegen 產生 Go server 介面與型別 |
| gRPC | Protocol Buffers |
| GraphQL 的適用與代價 |  |
| Webhook 設計 |  |

### 2.6 認證授權進階

| 標題 | 說明 |
| --- | --- |
| OAuth 2.0 與 OIDC | OpenID Connect |
| Access／Refresh token |  |
| [JWT vs PASETO](/backend/l2/2-6-auth-advanced/jwt-vs-paseto/) | 能依驗證方是誰、能不能查儲存，在 opaque token、JWT 與 PASETO 之間選型，並說出換協定能消除與消除不了的風險。 |
| RBAC 與 ABAC |  |
| 服務間認證 |  |
| 秘密管理 |  |
| 帳號枚舉防護 | 登入與註冊的一致回應、dummy bcrypt 讓回應時間一致、無權限回 404 而非 403 |
| 登入失敗鎖定與 IP throttle | 計數放哪、以資料庫時間計窗、鎖定被濫用成 DoS 的取捨 |
| Session 存資料庫 | 只存 ID 的 SHA-256、閒置與絕對逾時、輪換與過期清理 |
| 不使用 Email 的帳號體系 | 員工編號註冊、管理員由外部秘密庫注入與同步 |

### 2.7 容器與交付

| 標題 | 說明 |
| --- | --- |
| Docker | image、layer、multi-stage build |
| docker compose | 建立本地環境 |
| CI/CD | GitHub Actions 流水線 |
| 部署策略 | 滾動、藍綠、金絲雀 |
| [功能旗標](/backend/l2/2-7-containers/feature-flags/) | 用 Go 寫出穩定分桶的功能旗標，做到先部署、後放量，出事時不重新部署就能關掉，並學會清理過期旗標。 |
| 設定與環境分離 | Twelve-Factor |

### 2.8 可觀測性基礎

| 標題 | 說明 |
| --- | --- |
| 日誌、指標、追蹤三支柱 |  |
| 結構化日誌 | correlation ID |
| Prometheus 與 Grafana |  |
| 健康檢查 | readiness/liveness |
| `pprof` 找瓶頸 | CPU 與記憶體瓶頸 |
| 告警的基本原則 |  |

### 2.9 小結專案

| 標題 | 說明 |
| --- | --- |
| 短網址服務 v2 | 加入 Redis 快取、非同步記錄點擊、Docker 化與 CI |

## L3 資深：設計能水平擴展、能容錯的系統

L3 的重心從「怎麼寫」轉到「怎麼設計與取捨」，語言退居配角，但每章都用 Go 實作一個小版本來驗證理解。

### 3.1 分散式系統基礎

| 標題 | 說明 |
| --- | --- |
| 為什麼分散式很難 | 網路不可靠、時鐘不同步、部分失敗 |
| CAP 定理與 PACELC |  |
| 一致性模型 | 強一致、最終一致、因果一致 |
| 時鐘與順序 | Lamport clock、vector clock |
| 分散式 ID 產生 | Snowflake 的原理、時鐘倒退與機器編號分配 |
| 複寫 | 主從、多主、無主 |
| [MariaDB 與 Galera 叢集](/backend/l3/3-1-distributed-basics/mariadb-galera/) | 比較 MariaDB 非同步主從與 Galera 多主叢集：資料何時算寫入成功、會不會掉資料、誰負責故障轉移，以及各自的代價。 |
| 分片與一致性雜湊 | sharding |
| 共識演算法 | Paxos 與 Raft |
| 領導者選舉與分散式鎖 |  |
| Quorum 讀寫 |  |

### 3.2 跨服務資料一致性

| 標題 | 說明 |
| --- | --- |
| 分散式交易為什麼不好用 | 2PC 的問題 |
| Saga 模式 | 編排式 vs 協作式 |
| Outbox 模式 |  |
| 事件溯源與 CQRS | Event Sourcing |
| 冒等與去重的完整設計 |  |
| 資料複製與 CDC |  |

### 3.3 服務架構

| 標題 | 說明 |
| --- | --- |
| 單體、模組化單體、微服務 | 何時該拆 |
| Go 模組化單體實作 | 模組邊界、`internal` 套件、跨模組只走介面、依賴方向的自動檢查 |
| 依領域拆服務 | DDD 基礎、bounded context |
| 服務間通訊 | 同步 vs 非同步 |
| 服務探索與設定中心 |  |
| API Gateway 與 BFF | （反向代理與 Nginx 見 3.4） |
| 事件驅動架構 |  |
| 乾淨架構與六角架構 | Go 的實務專案分層 |

### 3.4 流量入口：反向代理與負載平衡

| 標題 | 說明 |
| --- | --- |
| 反向代理基礎 | 正向 vs 反向代理、`proxy_pass`、`Host` 與 `X-Forwarded-*`、常見用途 |
| [Nginx 負載平衡 1/3 選型](/backend/l3/3-4-traffic-entry/nginx-lb-selection/) | 比較輪詢、least_conn、random two 與 hash，依流量與 session 需求選出分配方式。 |
| Nginx 負載平衡 2/3 可靠性 | 被動 vs 主動健康檢查、重試風險、keepalive、DNS 解析陷阱、單點故障（keepalived／VRRP） |
| Nginx 負載平衡 3/3 營運 | `X-Forwarded-For`、`real_ip`、`stream` 平衡 TCP、何時改用 HAProxy 等 |
| Nginx 路由 | `location` 比對優先順序、依 host 分流、`rewrite`、`proxy_pass` 結尾斜線陷阱 |
| Nginx TLS 終結 | 憑證與自動更新、HTTP 轉 HTTPS、HTTP/2、HSTS、連到上游的 TLS |
| Nginx proxy_cache | 快取鍵、失效與清除、`stale`、避免快取個人化內容（快取理論見 2.3） |
| Nginx 串流與 WebSocket | SSE、`Upgrade` 標頭、關閉緩衝、長連線逾時 |
| Nginx 限流與安全加固 | `limit_req`、`limit_conn`、IP 名單、隱藏版本與標頭（演算法見 2.5、3.5） |
| Nginx 逾時、緩衝與排錯 | `proxy_*_timeout`、`proxy_buffering`、worker 與連線數、502／504 診斷 |

### 3.5 韌性設計

| 標題 | 說明 |
| --- | --- |
| 逾時、重試、指數退避與 jitter |  |
| 斷路器、艱壁隔離 |  |
| 限流、降級、負載篡除 |  |
| 依賴故障時的部分可用 | 物件儲存掛掉時刪除仍成功、上傳下載回 503，哪些操作可降級、哪些必須失敗 |
| 重試風暴與級聯故障 |  |
| 故障注入與混沌工程入門 |  |

### 3.6 效能調校

| 標題 | 說明 |
| --- | --- |
| 效能的度量 | 延遲、吞吐、p99 |
| 基準測試 | `testing.B`、壓力測試 |
| Go 執行期 | GC、逃逸分析、記憶體配置 |
| pprof、trace 進階用法 |  |
| 資料庫層的調校 | 讀寫分離、分區表 |
| 列表 API 的效能驗收 | 一萬筆資料下索引、分頁與序列化成本，以 RPS 與 p95 為目標 |
| 將 Go 服務優化一個數量級 | 實作 |

### 3.7 資料底層原理

| 標題 | 說明 |
| --- | --- |
| 儲存引擎 | B+ 樹 vs LSM-tree |
| WAL 與崩潰恢復 |  |
| MVCC |  |
| 搜尋引擎入門 | 倒排索引、Elasticsearch |
| 時序資料庫與列式儲存 |  |
| 資料倉儲與串流處理概觀 |  |

### 3.8 雲原生

| 標題 | 說明 |
| --- | --- |
| Kubernetes 核心 | Pod、Deployment、Service、Ingress |
| 設定、秘密與彈性伸縮 |  |
| 服務網格的取捨 | service mesh |
| 基礎設施即程式碼 | Terraform |
| 雲端服務全貌 | 計算、儲存、網路、托管資料庫 |
| Serverless 的適用與限制 |  |

### 3.9 系統設計實戰

| 標題 | 說明 |
| --- | --- |
| 系統設計的思考框架 | 需求、估算、設計、取捨 |
| 容量估算 | back-of-the-envelope |
| 系統設計案例 | 短網址、限流器、通知系統、訂單系統、搜尋建議、動態消息流 |
| 短網址服務 v3 | 拆服務、上 K8s、加入完整可觀測性 |

## L4 應用系統：把常見的 SaaS 功能做對

L4 的重心從「基礎設施怎麼設計」轉到「產品功能怎麼做對」。檔案上傳、權限、審計、通知、支付、串流、搜尋是幾乎每個 SaaS 都會遇到的功能，看似只是業務邏輯，實際上每一個都踩得到前三層的概念：一致性、冒等、佇列、安全、成本。每節的結構相同：需求與邊界、資料模型、核心流程、失敗情境、安全與成本，最後用 Go 實作一個可跑的最小版本。

> 各節只講功能本身的設計；用到的底層概念（佇列、冒等、快取、一致性）連回 L2、L3，不再重講。

### 4.1 檔案上傳與儲存

| 標題 | 說明 |
| --- | --- |
| 上傳的三種做法 | 經過後端轉發、預簽名 URL 直傳物件儲存、分片與斷點續傳（tus、S3 multipart） |
| multipart 解析與串流 | `multipart/form-data`：限制大小、不把整個檔案讀進記憶體 |
| [資料與檔案同一個請求](/backend/l4/4-1-file-upload/upload-with-data/) | 讓一個 multipart 請求同時帶欄位與檔案，做到先驗證再落地、限制大小、失敗不留殘骸。 |
| [檔案驗證](/backend/l4/4-1-file-upload/file-validation/) | 用檔頭判定類型、伺服器決定副檔名、檔名只當顯示用，並替解碼與解壓設上限。 |
| 惡意檔案防護 | 病毒掃描、隔離區、不在主網域提供使用者上傳的內容 |
| 上傳後處理管線 | 縮圖、轉檔、擷取中繼資料，用佇列非同步化（承 2.4） |
| 中繼資料與物件分離 | 資料庫存什麼、物件儲存存什麼、孤兒檔案清理 |
| [跨系統一致性](/backend/l4/4-1-file-upload/cross-system-consistency/) | 在資料庫與物件儲存之間，用不可變 key、同交易待刪除表與冪等清除，換圖與刪檔時不留懸空紀錄。 |
| 物件儲存實務 | MinIO／S3 相容：bucket、object key 設計、生命週期、單機與叢集部署 |
| 下載與分發 | 簽名 URL、Range 請求、CDN、存取控制 |
| 後端串流下載 | 先授權再開串流、`Content-Disposition` 與檔名編碼、不用預簽名 URL 的取捨 |
| 儲存生命週期與成本 | 儲存層級、過期刪除、去重（content hash） |

### 4.2 權限系統：RBAC 到細粒度授權

2.6 講了 RBAC 與 ABAC 的概念，本節講在真實產品裡怎麼建。

| 標題 | 說明 |
| --- | --- |
| 權限資料模型 | user、role、permission、resource，以及為什麼權限不該寫死在程式碼 |
| 多租戶下的 RBAC | 組織、團隊、專案層級的角色繼承 |
| 資源層級授權 | 「能不能改這一筆」比「能不能改這種資料」難 |
| 團隊與成員 | 一個資源多個參與者（leader、member、admin）的資料模型與權限判定 |
| 關係型授權（ReBAC） | Google Zanzibar、OpenFGA 的思路 |
| Go 授權實作 | 授權 middleware、Casbin、policy 與程式碼的分工 |
| 權限檢查的效能 | 快取、批次檢查、列表頁的過濾（授權條件下推到 SQL） |
| 權限變更的即時性 | 快取失效、token 內帶權限的代價 |
| 最小權限與角色爆炸 | 管理後台、角色爆炸（role explosion）與稽核 |

### 4.3 審計日誌

| 標題 | 說明 |
| --- | --- |
| 審計日誌與應用日誌的差異 | 目的、保存期限、讀者 |
| 審計日誌該記什麼 | who、what、when、where、before/after、request ID |
| 資料模型與寫入方式 | 同交易寫入 vs Outbox 非同步（承 3.2） |
| 失敗即整筆失敗 | 稽核與業務寫入同交易、被拒絕的嘗試也要記錄（與回滾的衝突怎麼處理） |
| 防竄改 | append-only、雜湊鏈、WORM 儲存 |
| 查詢與匯出 | 索引設計、分區、依租戶隔離 |
| 保存期限、隱私與刪除權 | 審計日誌如何與 GDPR 並存（承 5.2） |
| 敏感欄位處理 | 遮罩、不記密碼與 token |
| CDC 或 trigger 的取捨 | 用 CDC 或資料庫 trigger 做變更紀錄 |

### 4.4 通知系統

| 標題 | 說明 |
| --- | --- |
| 通知的種類與通道 | Email、簡訊、推播、站內信、Webhook |
| 通知服務的架構 | 事件進來、模板渲染、通道派送、狀態回報 |
| 通知模板與多語系 | 變數注入、預覽 |
| 使用者偏好與退訂 | 訂閱設定、退訂與合規（CAN-SPAM、GDPR） |
| 可靠投遞 | 重試、退避、死信、冒等去重，避免重複發送 |
| 第三方服務的失敗 | 供應商備援、頻率限制、成本控制 |
| 批次、摘要與延遲發送 | 合併通知、安靜時段、排程 |
| 送達追蹤 | 退信、投訴、開信與點擊事件的回收 |

### 4.5 支付與訂閱計費

| 標題 | 說明 |
| --- | --- |
| 支付流程全貌 | 授權、請款、退款、爭議；為什麼不自己碰信用卡（PCI 範圍） |
| 串接金流服務 | 例如 Stripe：Checkout、PaymentIntent、客戶端與伺服端分工 |
| 支付 Webhook 處理 | 簽章驗證、冒等、亂序與重送（承 2.5、3.2） |
| 訂單與支付狀態機 | 不可能的狀態轉換、補償與超時 |
| 金額的表示 | 整數最小單位、幣別、不能用 float |
| 帳本設計 | 雙式簿記、不可變分錄、餘額是推導出來的 |
| 對帳 | 與金流服務的差異偵測與修正 |
| 訂閱、試用與升降級 | 按比例計費（proration）與催收（dunning） |
| 用量計費 | 事件計量、聚合、與帳單週期對齊 |

### 4.6 串流與即時通訊

| 標題 | 說明 |
| --- | --- |
| 即時推送的選擇 | 短輪詢、長輪詢、SSE、WebSocket 的取捨 |
| SSE 與 WebSocket 實作 | Go 實作：心跳、斷線偵測、`Last-Event-ID` 續傳 |
| 連線管理 | 大量長連線的資源成本、優雅關機與重連風暴（承 2.1、3.5） |
| 多實例扇出 | 用 Redis Pub/Sub、NATS 或 Kafka 把訊息送到正確的連線 |
| 在線狀態與房間／頻道模型 | presence |
| 經過代理與負載平衡的陷阱 | 緩衝、逾時、黏性連線（承 3.4） |
| 影音串流基礎 | HLS／DASH、分段、轉碼、自適應位元率、DRM 概觀 |
| 檔案與影音的 Range 請求與 CDN |  |
| LLM 與長任務的 token 串流 | 進度回報與取消 |

### 4.7 搜尋

| 標題 | 說明 |
| --- | --- |
| 資料庫夠不夠用 | `LIKE`、PostgreSQL 全文檢索、何時才需要搜尋引擎 |
| 搜尋引擎的使用面 | Elasticsearch／OpenSearch 的索引、mapping、查詢 DSL（原理見 3.7） |
| 中文與多語言分詞 | 同義詞、拼字容錯 |
| 相關性排序與調校 | BM25、欄位加權、商業規則加權 |
| Facet、過濾、分頁與深分頁問題 |  |
| 自動完成與搜尋建議 |  |
| 搜尋索引資料同步 | 雙寫的問題、Outbox 與 CDC，確保搜尋索引與資料庫最終一致 |
| 零停機重建索引 | 別名切換與版本化 mapping |
| 搜尋的權限過濾與多租戶隔離 |  |

### 4.8 其他常見的 SaaS 功能

| 標題 | 說明 |
| --- | --- |
| 單一登入 | SAML、OIDC 企業 SSO、SCIM 帳號同步（承 2.6） |
| API 金鑰與開發者平台 | 金鑰雜湊儲存、範圍（scope）、輪替、用量配額 |
| 對外 Webhook 的發送端 | 簽章、重試、訂閱管理、投遞紀錄（接收端見 2.5） |
| 匯入與匯出 | 大型 CSV 的串流處理、背景工作、進度與錯誤報告 |
| Excel 匯出 | excelize、串流寫入大量列、公式與 CSV／Excel injection 防護 |
| 統計儀表板 | 聚合查詢、預先計算 vs 即時計算、時間分桶、索引與快取取捨 |
| 背景工作與工作流程 | 排程、長流程、可恢復（Temporal、Asynq 一類工具的取捨） |
| 軟刪除、回收桶與資料保存 | 刪除到底是什麼意思 |
| 功能開關與配額 | 方案（plan）與 entitlement 的檢查點 |
| 時區、多語系與貨幣 | 資料怎麼存、怎麼顯示 |
| 多租戶設計的應用面 | 資料隔離策略與 noisy neighbor（架構面見 5.3） |

### 4.9 小結專案

| 標題 | 說明 |
| --- | --- |
| 短網址服務 v4 | 做成 SaaS：權限 4.2、匯入 4.1／4.8、審計 4.3、通知 4.4、計費 4.5、串流 4.6、搜尋 4.7 |

## L5 架構與 Staff：為方向負責

L5 談的不再是單一系統，而是如何讓多個系統、多個團隊長期運作得好，並做出站得住腳的技術決策。

### 5.1 可靠性工程

| 標題 | 說明 |
| --- | --- |
| SLI、SLO、SLA 與錯誤預算 |  |
| 值班、事故處理流程 |  |
| 無責任事後審查 | postmortem |
| 容量規劃與成長預測 |  |
| 災難復原 | RPO、RTO、多區域部署 |
| 資料備份與還原演練 |  |
| 變更管理與功能開關 |  |

### 5.2 安全與合規

| 標題 | 說明 |
| --- | --- |
| 威脅建模 |  |
| OWASP Top 10 在後端的對應 |  |
| 加密 | 傳輸中、靜態、金鑰管理 |
| 零信任與網路分區 |  |
| 供應鏈安全 | 相依套件掃描 |
| 個資與隱私法規基礎 | 例如 GDPR |
| 審計日誌 | （實作見 4.3） |

### 5.3 資料與平台

| 標題 | 說明 |
| --- | --- |
| 資料管線 | ETL、串流、批次 |
| 資料湖與資料倉儲 |  |
| 內部開發者平台 | Platform Engineering |
| 多租戶設計 |  |
| 大型資料遠移與零停機遷移 |  |

### 5.4 成本與商業思維

| 標題 | 說明 |
| --- | --- |
| 雲端成本結構與 FinOps |  |
| 自建 vs 買服務 |  |
| 技術債的管理 |  |
| 把技術指標轉成商業語言 |  |

### 5.5 技術決策與團隊

| 標題 | 說明 |
| --- | --- |
| 架構決策紀錄 | ADR |
| 技術選型的評估框架 |  |
| 寫設計文件與 RFC |  |
| Code review 與工程文化 |  |
| 帶人、跨團隊溝通與影響力 |  |
| 產品化思維 | 從問題出發，而不是從技術出發 |

### 5.6 終點專題：真實世界案例讀法

| 標題 | 說明 |
| --- | --- |
| 如何讀一篇公司技術部落格 |  |
| 源碼閱讀方法 | 從 Go 標準庫與 etcd、Redis、Kubernetes 開始 |
| 持續學習的路線圖 |  |

## 全書專案：短網址服務的五次演進

同一個短網址服務貫穿全書，每一版都因為「上一版擁擠」而引入新概念，讓讀者親眼看到每個技術為什麼出現。

| 版本 | 上一版的痛點 | 新增能力 | 主要概念 | 對應章節 |
| --- | --- | --- | --- | --- |
| v1 | 從零開始 | 建立、查詢、重導向 API，資料存 SQL | HTTP、JSON、SQL、測試 | L1 |
| v2 | 讀取量變大、資料庫吃緊、點擊記錄拖慢回應 | Redis 快取、非同步記錄點擊、限流、Docker 化 | 快取模式、佇列、並行、CI/CD | L2 |
| v3 | 單一服務難以擴展與獨立部署 | 拆成寫入、讀取、分析三服務，上 K8s，完整可觀測性 | 服務拆分、冒等、韌性、分片 | L3 |
| v4 | 使用者要團隊協作、付費與企業級功能 | 團隊與權限、批次匯入、審計日誌、通知、方案計費、即時點擊串流、連結搜尋 | 檔案上傳、RBAC、審計、支付、SSE、搜尋 | L4 |
| v5 | 全球使用者、高可用要求 | 多區域部署、SLO 與告警、災難演練、成本最佳化 | 可靠性、安全、FinOps、ADR | L5 |
