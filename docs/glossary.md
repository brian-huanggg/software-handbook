# 術語表

全書統一用語。寫文章時照這張表使用；遇到新術語就加一列，依英文字母排序。

| English | 本書用法 | 不要用 | 備註 |
| --- | --- | --- | --- |
| admission control | admission control | 准入控制 | 不翻譯；Kubernetes 在資源建立前攔截並驗證的機制，可用來拒絕未簽章映像 |
| ambient authority | 環境權限 | 環境授權 | 憑證由瀏覽器依目的地自動附上、而非請求自己帶著；CSRF 濫用的對象 |
| asynchronous replication | 非同步複寫 | | 主節點寫完就回覆，副本稍後才追；MariaDB 標準版的預設複寫方式 |
| attack surface | 攻擊面 | 攻擊表面 | 攻擊者可利用的入口與工具總和；映像裡多一個 shell 或套件管理器就多一份 |
| blast radius | 影響範圍 | 爆炸半徑 | 一次出錯會波及的使用者與系統範圍；功能旗標與分階段發布用來縮小它 |
| blast radius | 影響範圍 | 爆炸半徑 | 一次出錯會波及的使用者與系統範圍；功能旗標與分階段發布用來縮小它 |
| bulkhead | 艙壁隔離 | 隔板 | |
| cache stampede | 快取擊穿 | | 熱門 key 失效瞬間大量請求打到資料庫 |
| capabilities | capabilities | 權能、能力 | Linux 把 root 權力拆成的細項（如 `NET_ADMIN`）；不翻譯 |
| certification | certification | 認證、驗證 | 不翻譯；Galera 各節點用同樣規則檢查 write-set 是否與已 commit 的交易衝突 |
| circuit breaker | 斷路器 | 熔斷器 | 指標惡化時自動收緊規則；審查的斷路器例子是 revert 率暴增時把低風險升為中風險（本書的示意設計） |
| circuit breaker | 斷路器 | 熔斷器 | 指標惡化時自動收緊規則；審查的斷路器例子是 revert 率暴增時把低風險升為中風險（本書的示意設計） |
| clustered index | 叢集索引 | 聚簇索引 | 表本身就是依主鍵排序的 B+ 樹，資料列在葉節點；InnoDB 的預設儲存方式 |
| code review | 程式碼審查 | 代碼審查、程式碼評審 | 合併前由人或工具檢查改動；AI 時代拆成多層驗證，人只看高風險與方向 |
| code review | 程式碼審查 | 代碼審查、程式碼評審 | 合併前由人或工具檢查改動；AI 時代拆成多層驗證，人只看高風險與方向 |
| consistent hashing | 一致性雜湊 | 一致性哈希 | 把 server 放上雜湊環，增減一台只影響相鄰的 key；nginx 用 `hash ... consistent` |
| container | 容器 | 貨櫃 | 共用主機核心、靠 namespace 與 cgroup 隔離的行程 |
| Content Security Policy (CSP) | 內容安全政策（CSP） | | 伺服器以標頭告訴瀏覽器哪些來源的腳本可以執行；nonce 型政策只放行帶本次回應 nonce 的腳本。第一次出現寫全名，之後只寫 CSP |
| cross-origin | 跨源 | 跨域 | 「域」會和網域混淆；跨源連 scheme 與 port 不同都算 |
| Cross-Origin Resource Sharing (CORS) | 跨源資源共用（CORS） | 跨域資源共享 | 第一次出現寫全名，之後只寫 CORS |
| CORS misconfiguration | CORS 錯誤設定 | CORS 配置錯誤 | 把 `Access-Control-Allow-Origin` 開給不該信任的 origin |
| cross-site request forgery (CSRF) | 跨站請求偽造（CSRF） | 跨站請求攻擊 | 讓受害者已登入的瀏覽器送出非本意的請求；第一次出現寫全名，之後只寫 CSRF |
| cross-site scripting (XSS) | 跨站腳本（XSS） | 跨站指令碼 | 攻擊者讓自己的 JavaScript 在目標網站的頁面裡執行；第一次出現寫全名，之後只寫 XSS |
| digest | digest | 摘要 | 不翻譯；映像內容的 SHA-256 雜湊，`@sha256:...` 固定內容 |
| distroless image | distroless 映像 | 無發行版映像 | 只含應用程式與執行期相依，無 shell 與套件管理器 |
| DOM-based XSS | DOM 型 XSS | | 酬載由前端 JavaScript 從來源寫進出口（如 `innerHTML`）；`#` 之後的片段不送到伺服器，伺服器看不到 |
| double-submit cookie | double-submit cookie | 雙重提交 Cookie | 不翻譯；token 同時放 Cookie 與表單、比對兩者一致；未簽章、未綁 session 時可被繞過 |
| failover | 故障轉移 | 容錯移轉、故障切換 | 主節點失效後改由另一個節點接手 |
| feature flag | 功能旗標 | 特性開關 | 不重新部署就能開關新行為的設定；用來縮小影響範圍與快速回滾 |
| feature flag | 功能旗標 | 特性開關 | 不重新部署就能開關新行為的設定；用來縮小影響範圍與快速回滾 |
| Fetch Metadata | Fetch Metadata | 擷取中繼資料 | 不翻譯；瀏覽器附上的 `Sec-Fetch-*` header，如 `Sec-Fetch-Site` 標明請求是否跨站，網頁腳本無法偽造 |
| flow control | flow control | 流量控制 | 不翻譯；Galera 節點套用太慢時，通知其他節點暫停寫入 |
| Galera cluster | Galera 叢集 | Galera 集群 | MariaDB 的多主叢集，用 certification-based 複寫 |
| Golden Image | Golden Image | 黃金映像 | 平台團隊建置、掃描、簽章並維護的核准 base image；組織內服務一律從它開始 |
| goroutine | goroutine | 協程、Go 程 | 不翻譯 |
| HttpOnly | HttpOnly | | 不翻譯；Cookie 屬性，禁止 `document.cookie` 讀取；擋不住腳本以使用者身分送出請求 |
| idempotency | 冪等 | 冒等、幂等 | |
| image | 映像 | 鏡像、鏡像檔 | 唯讀 layer 加啟動設定；container 由它啟動 |
| incremental state transfer (IST) | IST | | 節點重新加入時只補缺少的 write-set；第一次出現寫全名 |
| insecure direct object reference (IDOR) | IDOR（不安全的直接物件參照） | 不安全直接對象引用 | 只靠 ID 取資料、不檢查呼叫者是否有權限；第一次出現寫全名，之後只寫 IDOR |
| integration test | 整合測試 | 集成測試 | 讓程式碼接上真的資料庫、快取等依賴一起跑的測試；與只用 fake 的單元測試相對 |
| layer | layer | 層、圖層 | 不翻譯；Dockerfile 每個 `COPY`／`RUN` 產生一個，建好後不再改動 |
| layered verification | 分層驗證 | | 把審查拆成多道各自攔截不同風險的檢查，再依風險加上人類核准 |
| layered verification | 分層驗證 | | 把審查拆成多道各自攔截不同風險的檢查，再依風險加上人類核准 |
| load balancer | 負載平衡器 | 負載均衡器 | 站在多個後端前面，決定請求交給哪一台 |
| load shedding | 負載卸除 | 負載篡除 | |
| login CSRF | 登入 CSRF | | 偽造登入請求，讓受害者在不知情下登入攻擊者的帳號 |
| merge queue | 合併佇列 | 合併隊列 | 合併前在最新 main 上重跑檢查，避免兩個各自通過的 PR 合在一起才壞 |
| merge queue | 合併佇列 | 合併隊列 | 合併前在最新 main 上重跑檢查，避免兩個各自通過的 PR 合在一起才壞 |
| migration | 遷移（資料庫結構版本控制時寫 migration） | 遠移 | |
| multi-stage build | multi-stage build | 多階段建置 | 不翻譯；Builder 階段建置，Production 階段只複製產物 |
| mutation XSS (mXSS) | 突變型 XSS | | 經 sanitizer 處理過的 HTML 在瀏覽器重新解析時改變結構，酬載因此復活 |
| nonce | nonce | 隨機數 | 不翻譯；CSP 中每個回應都重新產生的隨機值，只有帶它的 `<script>` 才能執行 |
| opaque origin | 不透明 origin | | 序列化為 `null`，與任何 origin 都不相同 |
| origin | origin | 來源、源 | 不翻譯；指 scheme、host、port 三者組成的組合 |
| output encoding | 輸出編碼 | 輸出轉義 | 資料放進頁面時，依所在位置（HTML 本文、屬性、URL、JavaScript）做對應的跳脫；XSS 的主要修法 |
| page split | 頁分裂 | 頁面分割 | B+ 樹的目標頁已滿時切成兩頁，兩頁各剩約一半；隨機主鍵會頻繁發生 |
| payload | 酬載 | 載荷 | 攻擊者送進目標、要讓它執行的內容；在 XSS 中是那段注入的標記或腳本 |
| preflight request | 預檢請求 | 預檢、OPTIONS 請求 | 瀏覽器在非簡單請求前自動發出的 `OPTIONS` |
| primary / replica | primary／replica | master／slave、主庫／從庫 | 不翻譯；標題或總稱可寫「主從複寫」 |
| quorum | quorum | 法定人數 | 不翻譯；超過一半節點互相連得上，只有這一邊能繼續寫入 |
| recovery point objective (RPO) | RPO | | 故障後最多可接受丟失的資料量；RPO = 0 表示不丟已確認的寫入；第一次出現寫全名 |
| reflected XSS | 反射型 XSS | | 酬載在請求裡，伺服器立刻把它放進回應；要受害者點擊連結才發生 |
| replication | 複寫 | 複製、同步 | 把資料變更送到其他節點 |
| risk tier | 風險分級 | 風險等級 | 由改動的內容（而非作者）決定需要 0、1 或 2 位人類核准 |
| risk tier | 風險分級 | 風險等級 | 由改動的內容（而非作者）決定需要 0、1 或 2 位人類核准 |
| rollback | 回滾 | 回退 | 把已上線的改動撤回（關旗標或 revert），分層驗證的最後一道安全網 |
| rollback | 回滾 | 回退 | 把已上線的改動撤回（關旗標或 revert），分層驗證的最後一道安全網 |
| Ryuk | Ryuk | | 不翻譯；Testcontainers 啟動的小容器，測試程序結束（含被強制終止）後負責清掉這次啟動的容器 |
| same-origin policy | 同源政策 | 同源策略 | |
| same-site | 同站 | | 只看 scheme 與可註冊網域，不看子網域與 port |
| SameSite | SameSite | 同站屬性 | 不翻譯；Cookie 屬性，限制跨站請求是否附上該 Cookie（`Strict`、`Lax`、`None`） |
| sanitizer | sanitizer | 清洗器 | 不翻譯；把使用者貼的 HTML 去掉危險部分、保留安全標籤的函式庫（如 DOMPurify、bluemonday）；不要自己寫 |
| secondary index | 次要索引 | 二級索引、輔助索引 | 非主鍵的索引；InnoDB 的葉節點存主鍵值，Postgres 存 ctid |
| semi-synchronous replication | 半同步複寫 | | 主節點等至少一個副本收到才回覆；逾時會退回非同步 |
| session persistence | session 保持 | 會話黏滯、黏性會話 | 讓同一個使用者的請求固定到同一台後端；nginx 用 `ip_hash` 或 `hash` 實現 |
| simple request | 簡單請求 | | 不需預檢的跨源請求，條件見後端 1.4 CORS |
| sink | 出口 | 匯 | 前端程式中會把字串當成程式或標記處理的位置（如 `innerHTML`、`eval`）；DOM 型 XSS 從來源流到這裡 |
| smooth weighted round robin | 平滑加權輪詢 | | 每次把 weight 加到分數、挑最高分、從贏家扣總 weight，讓重的 server 不連續被選 |
| Snowflake ID | Snowflake ID | 雪花 ID、雪花演算法 | 不翻譯；64 bits：時間、機器編號、序號，有序且可分散產生 |
| software bill of materials (SBOM) | 軟體物料清單（SBOM） | 軟體材料清單 | 列出映像內所有元件與版本；第一次出現寫全名，之後只寫 SBOM |
| source | 來源 | | 前端程式中攻擊者可控制的輸入位置（如 `location.hash`、`postMessage`）；與 sink 相對 |
| split-brain | split-brain | 腦裂 | 不翻譯；叢集分裂後兩邊都以為自己是主節點 |
| state snapshot transfer (SST) | SST | | 節點重新加入時複製整份資料；第一次出現寫全名 |
| stored XSS | 儲存型 XSS | | 酬載先被存進伺服器，之後在別人看到那份資料時執行；最危險的一種 |
| supply chain attack | 供應鏈攻擊 | 供應商攻擊 | 攻擊你所依賴的映像、套件或建置流程，而非直接攻擊你 |
| synchronizer token | CSRF token | 同步令牌 | 伺服器發、綁 session 的不可猜測值，不安全方法的請求必須帶上並由伺服器驗證 |
| tag | tag | 標籤 | 不翻譯；映像上可被改指的名稱，與 digest 相對 |
| Testcontainers | Testcontainers | | 不翻譯；在測試程式裡透過 Docker 啟動一次性真實服務的函式庫，Go 版是 testcontainers-go |
| thread | 執行緒 | 線程 | |
| timeout | 逾時 | 超時 | |
| Trusted Types | Trusted Types | | 不翻譯；瀏覽器機制，強制 `innerHTML` 等出口只接受經指定函式處理的值，用來根治 DOM 型 XSS |
| upstream | upstream | 上游 | 不翻譯；nginx 中一組後端 server 的群組，用 `proxy_pass` 指向它 |
| UUID | UUID | 通用唯一識別碼 | 128 bits；v4 全隨機、v7 前 48 bits 是毫秒時間戳（RFC 9562） |
| virtually synchronous | virtually synchronous | 虛擬同步 | 不翻譯；已 commit 的交易保證會被每個節點套用，但不保證同一時刻 |
| wait strategy | 等待策略 | 等待機制 | 容器啟動後持續檢查某個條件（log、port、SQL 連線）、成立才算就緒；用來取代 `time.Sleep` |
| weighted round robin | 加權輪詢 | 加權循環 | 依 weight 比例輪流分配請求；nginx upstream 的預設方式 |
| whiteout | whiteout | 白除、刪除標記 | 不翻譯；後面 layer 以 `.wh.` 前綴檔標示「這個檔案已刪除」，舊 layer 內容不變 |
| write-set | write-set | 寫入集 | 不翻譯；一筆交易要複寫給其他節點的變更內容 |
