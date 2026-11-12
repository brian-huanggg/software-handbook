# 術語表

全書統一用語。寫文章時照這張表使用；遇到新術語就加一列，依英文字母排序。

| English | 本書用法 | 不要用 | 備註 |
| --- | --- | --- | --- |
| admission control | admission control | 准入控制 | 不翻譯；Kubernetes 在資源建立前攔截並驗證的機制，可用來拒絕未簽章映像 |
| attack surface | 攻擊面 | 攻擊表面 | 攻擊者可利用的入口與工具總和；映像裡多一個 shell 或套件管理器就多一份 |
| bulkhead | 艙壁隔離 | 隔板 | |
| cache stampede | 快取擊穿 | | 熱門 key 失效瞬間大量請求打到資料庫 |
| capabilities | capabilities | 權能、能力 | Linux 把 root 權力拆成的細項（如 `NET_ADMIN`）；不翻譯 |
| clustered index | 叢集索引 | 聚簇索引 | 表本身就是依主鍵排序的 B+ 樹，資料列在葉節點；InnoDB 的預設儲存方式 |
| container | 容器 | 貨櫃 | 共用主機核心、靠 namespace 與 cgroup 隔離的行程 |
| cross-origin | 跨源 | 跨域 | 「域」會和網域混淆；跨源連 scheme 與 port 不同都算 |
| Cross-Origin Resource Sharing (CORS) | 跨源資源共用（CORS） | 跨域資源共享 | 第一次出現寫全名，之後只寫 CORS |
| CORS misconfiguration | CORS 錯誤設定 | CORS 配置錯誤 | 把 `Access-Control-Allow-Origin` 開給不該信任的 origin |
| digest | digest | 摘要 | 不翻譯；映像內容的 SHA-256 雜湊，`@sha256:...` 固定內容 |
| distroless image | distroless 映像 | 無發行版映像 | 只含應用程式與執行期相依，無 shell 與套件管理器 |
| Golden Image | Golden Image | 黃金映像 | 平台團隊建置、掃描、簽章並維護的核准 base image；組織內服務一律從它開始 |
| goroutine | goroutine | 協程、Go 程 | 不翻譯 |
| idempotency | 冪等 | 冒等、幂等 | |
| image | 映像 | 鏡像、鏡像檔 | 唯讀 layer 加啟動設定；container 由它啟動 |
| insecure direct object reference (IDOR) | IDOR（不安全的直接物件參照） | 不安全直接對象引用 | 只靠 ID 取資料、不檢查呼叫者是否有權限；第一次出現寫全名，之後只寫 IDOR |
| layer | layer | 層、圖層 | 不翻譯；Dockerfile 每個 `COPY`／`RUN` 產生一個，建好後不再改動 |
| load shedding | 負載卸除 | 負載篡除 | |
| migration | 遷移（資料庫結構版本控制時寫 migration） | 遠移 | |
| multi-stage build | multi-stage build | 多階段建置 | 不翻譯；Builder 階段建置，Production 階段只複製產物 |
| opaque origin | 不透明 origin | | 序列化為 `null`，與任何 origin 都不相同 |
| origin | origin | 來源、源 | 不翻譯；指 scheme、host、port 三者組成的組合 |
| page split | 頁分裂 | 頁面分割 | B+ 樹的目標頁已滿時切成兩頁，兩頁各剩約一半；隨機主鍵會頻繁發生 |
| preflight request | 預檢請求 | 預檢、OPTIONS 請求 | 瀏覽器在非簡單請求前自動發出的 `OPTIONS` |
| same-origin policy | 同源政策 | 同源策略 | |
| same-site | 同站 | | 只看 scheme 與可註冊網域，不看子網域與 port |
| secondary index | 次要索引 | 二級索引、輔助索引 | 非主鍵的索引；InnoDB 的葉節點存主鍵值，Postgres 存 ctid |
| simple request | 簡單請求 | | 不需預檢的跨源請求，條件見後端 1.4 CORS |
| Snowflake ID | Snowflake ID | 雪花 ID、雪花演算法 | 不翻譯；64 bits：時間、機器編號、序號，有序且可分散產生 |
| software bill of materials (SBOM) | 軟體物料清單（SBOM） | 軟體材料清單 | 列出映像內所有元件與版本；第一次出現寫全名，之後只寫 SBOM |
| supply chain attack | 供應鏈攻擊 | 供應商攻擊 | 攻擊你所依賴的映像、套件或建置流程，而非直接攻擊你 |
| tag | tag | 標籤 | 不翻譯；映像上可被改指的名稱，與 digest 相對 |
| thread | 執行緒 | 線程 | |
| timeout | 逾時 | 超時 | |
| UUID | UUID | 通用唯一識別碼 | 128 bits；v4 全隨機、v7 前 48 bits 是毫秒時間戳（RFC 9562） |
| whiteout | whiteout | 白除、刪除標記 | 不翻譯；後面 layer 以 `.wh.` 前綴檔標示「這個檔案已刪除」，舊 layer 內容不變 |
