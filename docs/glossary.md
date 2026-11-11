# 術語表

全書統一用語。寫文章時照這張表使用；遇到新術語就加一列，依英文字母排序。

| English | 本書用法 | 不要用 | 備註 |
| --- | --- | --- | --- |
| bulkhead | 艙壁隔離 | 隔板 | |
| cache stampede | 快取擊穿 | | 熱門 key 失效瞬間大量請求打到資料庫 |
| cross-origin | 跨源 | 跨域 | 「域」會和網域混淆；跨源連 scheme 與 port 不同都算 |
| Cross-Origin Resource Sharing (CORS) | 跨源資源共用（CORS） | 跨域資源共享 | 第一次出現寫全名，之後只寫 CORS |
| CORS misconfiguration | CORS 錯誤設定 | CORS 配置錯誤 | 把 `Access-Control-Allow-Origin` 開給不該信任的 origin |
| goroutine | goroutine | 協程、Go 程 | 不翻譯 |
| idempotency | 冪等 | 冒等、幂等 | |
| load shedding | 負載卸除 | 負載篡除 | |
| migration | 遷移（資料庫結構版本控制時寫 migration） | 遠移 | |
| opaque origin | 不透明 origin | | 序列化為 `null`，與任何 origin 都不相同 |
| origin | origin | 來源、源 | 不翻譯；指 scheme、host、port 三者組成的組合 |
| preflight request | 預檢請求 | 預檢、OPTIONS 請求 | 瀏覽器在非簡單請求前自動發出的 `OPTIONS` |
| same-origin policy | 同源政策 | 同源策略 | |
| same-site | 同站 | | 只看 scheme 與可註冊網域，不看子網域與 port |
| simple request | 簡單請求 | | 不需預檢的跨源請求，條件見後端 1.4 CORS |
| thread | 執行緒 | 線程 | |
| timeout | 逾時 | 超時 | |
