---
title: Nginx 負載平衡 1/3：選型與演算法
description: 比較輪詢、least_conn、random two 與 hash，依流量與 session 需求選出分配方式。
sidebar:
  order: 6
---

## 先想想

短網址服務 v3 的讀取服務負責 `GET /{code}` 轉址，跑在三台機器上，前面是 nginx，設定只有預設的 round-robin。其中一台是舊機器，處理速度只有另外兩台的一半。

流量上來之後，舊機器的 CPU 先滿載，p99 延遲從 40 ms 變成 900 ms（示意數字），另外兩台卻還有餘裕。另一個問題在管理後台：登入 session 存在實例的記憶體裡，使用者每按幾下就被登出。

同事說：「前者調 weight，後者加一行 `ip_hash`，讓同一個人固定打同一台。」weight 要怎麼設？`ip_hash` 真的能解決登出問題嗎，有什麼隱憂？

<details>
<summary>提示</summary>

想想 `ip_hash` 看到的「同一個人」是什麼：是使用者的裝置，還是使用者所在網路的出口？

</details>

## 是什麼

**負載平衡器（load balancer）** 是站在多個後端實例前面的轉發點，替每個請求或連線決定交給哪一台。Nginx 的做法是把後端列成一個 **upstream** 群組，再用 `proxy_pass` 指向它。

Nginx 可以工作在兩層：`http` 模組是第 7 層（L7），看得懂 HTTP；`stream` 模組是第 4 層（L4），只轉發 TCP 或 UDP。兩者共用同一套 upstream 語法，差別在於 nginx 看得到多少東西。

本篇只回答「下一個請求交給哪一台」，也就是選型。後端壞掉怎麼辦、請求失敗要不要重試，在第 2 篇；客戶端真實 IP 與 nginx 以外的選擇，在第 3 篇。

## 為什麼存在

沒有負載平衡器時，客戶端必須自己知道所有實例的位址。最常見的替代做法是 DNS 回傳多筆 A 紀錄，但 DNS 有快取：實例掛了，TTL 內仍有客戶端被導向它（TTL 設 60 秒，就是最多一分鐘的錯誤）。DNS 也不知道每台機器的能力不同，只會平均分。

用數字看第二個問題。假設 v3 讀取流量是 5,000 rps，兩台新機器各能處理 2,500 rps，舊機器 1,250 rps（皆為估算）。平均分配時每台約 1,667 rps：

- 舊機器：1,667 / 1,250 ≈ 133%，過載。
- 新機器：1,667 / 2,500 ≈ 67%，閒置。

按能力 2:2:1 分配，三台各是 2,000、2,000、1,000 rps，利用率都是 80%。總容量沒有變，只是分配方式讓它能被用完。負載平衡器就是把這個分配決定，從客戶端移到一個你能控制的地方，加減機器時只改 upstream 再 reload。

## 怎麼運作

:::note
本篇的範例在 `playground/golang/backend/l3/3-4-traffic-entry/nginx-lb-selection/`。Go 程式用記憶體重做 nginx 的選擇演算法，不轉發任何流量。設定檔在 `conf/nginx.conf`
:::

### 先分 L4 與 L7

```text
L7 (http)    client ──HTTP 請求──► nginx ──挑一台──► backend    每個請求各選一次
L4 (stream)  client ──TCP 連線───► nginx ──挑一台──► backend    每條連線只選一次
```

看右邊的註解：L7 每個請求都重新選，L4 在連線建立時選一次，之後那條連線的所有位元組都走同一台。

L7 解析 HTTP，所以能依 `Host`、URI、header、cookie 決定去向，也能讓許多客戶端的請求共用幾條對後端的連線。代價是要花 CPU 解析，且 TLS 通常在 nginx 終止。

L4 不解析協定，開銷小，適合 MariaDB 這類不是 HTTP 的 TCP 服務，例如分散到 Galera 叢集的三個節點：

```nginx title="nginx.conf"
stream {
    upstream db_nodes {
        zone db_nodes 64k;
        least_conn;
        server 10.0.2.11:3306;
        server 10.0.2.12:3306;
        server 10.0.2.13:3306;
    }

    server {
        listen 3306;
        proxy_pass db_nodes;
    }
}
```

兩件事要記住。

1. 第一，後端在 L4 看到的來源是 nginx 的位址，不是客戶端，要拿到真實 IP 需要另外設定（第 3 篇）。
2. 第二，因為只在連線建立時選擇，一條承載很多請求的長連線（HTTP/2、gRPC）會整條黏在同一台，後面的常見坑會再看到。

### 加權輪詢：預設的選擇

不寫任何演算法指令時，nginx 用加權輪詢（weighted round robin）。回到短網址 v3，把舊機器的 weight 設為其他兩台的一半：

```nginx title="nginx.conf"
    upstream read_weighted {
        zone read_weighted 64k;
        server 10.0.1.11:8080 weight=2;
        server 10.0.1.12:8080 weight=2;
        server 10.0.1.13:8080 weight=1;
    }
```

weight 5:1:1 的天真做法是連續送五個給 a，再各送一個給 b、c，這會讓 a 瞬間吃到一整段流量。Nginx 用的是平滑加權輪詢（smooth weighted round robin）：每次先把每台的 weight 加到它的分數，挑分數最高的，再從贏家的分數扣掉總 weight。

```go title="balancer.go"
func smoothPick(servers []*Server) *Server {
	var best *Server
	total := 0
	for _, s := range servers {
		s.current += s.Weight
		total += s.Weight
		if best == nil || s.current > best.current {
			best = s
		}
	}
	if best == nil {
		return nil
	}
	best.current -= total
	return best
}
```

以 5:1:1 手算前三次：分數從 (0, 0, 0) 開始，第 1 次加完是 (5, 1, 1)，a 贏，扣 7 變 (-2, 1, 1)；第 2 次 (3, 2, 2)，a 贏，變 (-4, 2, 2)；第 3 次 (1, 3, 3)，b 與 c 同分，先出現的 b 贏。這樣 a 的連續次數被打散：

```bash
go run .
```

```text
== round robin, weights a=5 b=1 c=1 ==
first 7 requests: a a b a c a a
```

### 依負載選：least_conn 與 random two

加權輪詢假設每個請求的成本差不多。如果有的請求要 5 ms，有的是 3 秒的上傳或慢查詢，輪詢就會讓某台碰巧累積一堆慢請求。

`least_conn` 改看目前處理中的連線數：選「連線數除以 weight」最小的那台，平手時用加權輪詢輪流。範例模擬 3,000 個請求，每 tick 進來一個；快機器處理一個請求要 2 tick，慢機器要 20 tick：

```text
== slow server: round robin vs least_conn vs random two ==
round_robin  fast-1=1000 fast-2=1000 slow=1000
least_conn   fast-1=1428 fast-2=1429 slow= 143
random two   fast-1=1389 fast-2=1372 slow= 239
```

輪詢讓慢機器收到 1/3 的請求，每個都要卡 20 tick，連線一直堆積。`least_conn` 讓它只收到約 5%，因為它的在途連線數很快就比別人高。

`random two` 是 `random two least_conn` 的簡稱：隨機挑兩台（依 weight），再選其中連線較少的。看起來比 `least_conn` 粗糙，但它在多個 nginx 同時做決定時更穩。原因是每個 nginx 看到的「最空的那台」是同一台，大家同時衝過去，這台瞬間被灌爆（群聚效應，herd behavior）；隨機挑兩台讓各個 nginx 的選擇分散開。連續請求之間只比兩台，也比掃描整個清單便宜。

```nginx title="nginx.conf"
    upstream read_random_two {
        zone read_random_two 64k;
        random two least_conn;
        server 10.0.1.11:8080;
        server 10.0.1.12:8080;
        server 10.0.1.13:8080;
    }
```

`least_time`（選平均回應時間最短的）與 `sticky` 是 NGINX Plus 商業版功能，開源版沒有。

### 要「同一人同一台」：hash 與 session 保持

有些情況下，請求不該隨便換台：session 存在實例記憶體、本機快取要命中。這需求叫 session 保持（session persistence）。開源版有兩種做法：

```nginx title="nginx.conf"
    upstream admin_ip_hash {
        ip_hash;
        server 10.0.3.11:8080;
        server 10.0.3.12:8080;
    }

    upstream admin_cookie {
        zone admin_cookie 64k;
        hash $cookie_sid consistent;
        server 10.0.3.11:8080;
        server 10.0.3.12:8080;
    }
```

`ip_hash` 用客戶端 IPv4 的前三個位元組（IPv6 則是完整位址）當 key，所以同一個 /24 網段的所有人會到同一台。`hash` 讓你自己指定 key，可以是 `$cookie_sid` 這類變數。

再來是 `consistent` 的差別。不加它，nginx 把 key 的雜湊對 server 數量取餘數，增減一台，大部分 key 都換台。加上它改用一致性雜湊（consistent hashing）：把 server 放在雜湊環上，key 順時針找到第一個 server，增減一台只影響環上相鄰的那一段。範例用 10,000 個使用者從 3 台增加到 4 台：

```text
== adding a 4th server: how many of 10,000 users change server ==
hash (mod):         7514 moved
hash consistent:    3280 moved
```

理想值分別是 7,500（取餘數，3 變 4 時約四分之三的 key 換台）與 2,500（一致性雜湊，新機器接走約四分之一）。實測偏高一些，是範例用的雜湊函式與 160 個虛擬點分得不夠均勻。

最後看 `ip_hash` 的「同一個人」：

```text
== ip_hash: 1,000 users behind one NAT vs 1,000 spread out ==
behind one NAT: b=1000
spread out:     a=341 b=327 c=332
```

1,000 個人從同一個公司 NAT 出來，共用 /24，全部落在 b；分散的使用者才會平均。

回到一開始的問題：舊機器過載，把它的 weight 設成另外兩台的一半（2:2:1），三台的利用率都是 80%；請求成本差異大時，再改成 `least_conn`。至於登出，`ip_hash` 並沒有真正解決：它看到的是網路出口，不是人，公司或學校的 NAT 後面所有人都會擠到同一台，反而製造新的熱點；後端重啟或縮容時，session 一樣會消失。要同一人同一台，用 `hash $cookie_sid consistent` 比 `ip_hash` 精確；但最穩的做法是把 session 搬出實例（例如 L2 學過的 Redis），讓任何一台都能處理任何請求。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 加權輪詢（預設） | 最簡單，結果可預測 | 不看實際負載，請求成本不均時會堆積 | 無狀態、請求耗時接近、機器規格不同 |
| `least_conn` | 自動避開忙碌的後端 | 需要 `zone` 才能跨 worker 共享計數；多個 nginx 時有群聚效應 | 請求耗時差異大：上傳、長輪詢、慢查詢 |
| `random two least_conn` | 多個 nginx 並存時比 `least_conn` 穩 | 結果不可預測，除錯較難 | 多台 nginx 前端、大量後端 |
| `ip_hash` | 一行設定，不需 cookie | 以網段為單位，NAT 與 CDN 後面會偏斜；後端數量變動就重分配 | 內部系統、客戶端 IP 分散且穩定 |
| `hash $cookie_sid consistent` | 以使用者識別碼為單位；增減後端只搬少數 key | 沒有 cookie 的請求都落在同一台；後端掛了仍會丟 session | 有 session 識別碼、又無法外部化 session |
| 外部 session 儲存（如 Redis） | 任何後端都能處理任何請求，最穩 | 多一個元件與一次網路往返 | 大多數有登入狀態的服務 |

什麼時候不該用 session 保持：

- **想靠它省掉 session 外部化**：保持只是降低換台的機率，實例重啟、縮容、被健康檢查剔除時，session 一樣會消失。要承受這些情況，session 就不能只放在某台的記憶體。
- **key 的分佈極不均**：`hash` 的 key 如果集中在少數值（例如大客戶的租戶 ID），那一台會被壓垮，別的台閒著。
- **後端需要隨流量快速增減**：每次增減都會搬動 key，連帶讓本機快取失效。

不該用 L4 的情況：需要依 URL 或 header 路由，或後端用 HTTP/2、gRPC 且需要把請求攤開時，要用 L7。

## 常見坑與故障案例

以下案例都是示意案例，沒有對應到特定公開事故。

### ip_hash 後面是公司 NAT 或 CDN

症狀：某個大客戶上線後，後台三台實例裡有一台 CPU 一直滿載，另外兩台幾乎沒事。原因：該客戶的幾百位員工從同一個 NAT 出口進來，`ip_hash` 只看前三個位元組，全部落在同一台。如果 nginx 前面還有一層 CDN 或負載平衡器，nginx 看到的來源就是那層的位址，同樣會偏斜。對策：改用 `hash $cookie_sid consistent` 或其他使用者層級的 key，或把 session 外部化；後者還需要先處理真實客戶端 IP（第 3 篇）。

### 沒設 zone，least_conn 在多個 worker 間不準

症狀：`worker_processes auto` 的 nginx 在 8 顆 CPU 的機器上，`least_conn` 的結果看起來跟輪詢差不多，慢的後端仍然堆積連線。原因：官方文件說明，`zone` 是讓 upstream 的執行期狀態在 worker 之間共享；沒有它，每個 worker 各自維護自己的狀態，各自只看到自己經手的那份連線。對策：每個要依負載選擇的 upstream 都加上 `zone`，大小夠放所有 server 即可。

### hash 沒加 consistent，一擴容全部換台

症狀：加了一台實例後，登入 session 與本機快取在幾分鐘內大量失效，使用者被登出，資料庫讀取量暴增。原因：不加 `consistent` 時，key 對 server 數量取餘數，3 變 4 台，前面範例顯示約 75% 的 key 換了台。對策：需要保持對應關係的 `hash`，一律加 `consistent`；要暫時下線一台時，標記 `down`，不要直接從清單刪掉，才能保留原本的雜湊結果。

### 沒有 cookie 的請求全部落到同一台

症狀：用 `hash $cookie_sid consistent` 後，使用者登入沒問題，但登入頁與健康檢查頁的流量全部打在同一台，那台比別台忙。原因：沒帶 cookie 的請求，`$cookie_sid` 是空字串，空字串的雜湊是固定值，全部指向同一個點。對策：只對需要保持的路徑（如 `/admin/`）使用 `hash` 的 upstream，其他路徑用輪詢或 `least_conn` 的 upstream。

### L4 加上長連線，新機器分不到流量

症狀：gRPC 服務用 `stream` 轉發，擴容加了第 4 台，它的 CPU 一個小時後仍接近 0，舊的三台卻很忙。原因：L4 只在連線建立時選一次，而 gRPC 用 HTTP/2，一條連線承載所有請求，客戶端的連線早已建立並黏在舊的三台上。對策：改用 L7（`http` 模組的 `grpc_pass`）讓每個請求各自選擇，或讓客戶端定期重連。

## 小練習

### 動手寫

範例的 `smoothPick` 是 nginx 預設輪詢的核心。請補完 `exercise/wrr.go` 的 `Next`：依本文的規則（每台加上 weight、挑分數最高的、從贏家扣掉總 weight）挑出下一台，清單為空時回傳 `nil`。

起始程式碼在 `playground/golang/backend/l3/3-4-traffic-entry/nginx-lb-selection/exercise/`。做對時，下面的指令會通過：

```bash
go test -tags exercise ./exercise/
```

<details>
<summary>參考解答</summary>

```go title="wrr.go"
func Next(servers []*Server) *Server {
	var best *Server
	total := 0
	for _, s := range servers {
		s.Current += s.Weight
		total += s.Weight
		if best == nil || s.Current > best.Current {
			best = s
		}
	}
	if best == nil {
		return nil
	}
	best.Current -= total
	return best
}
```

關鍵有兩個。比較用「大於」，同分時先出現的贏，所以 `a b a b` 的順序固定。扣的是「所有 server 的 weight 總和」，不是贏家自己的 weight，這才讓它的分數在一個週期後回到 0，週期內被均勻打散。

</details>

### 辨識

下面是 AI 產生的 Go 反向代理，目的是讓每個使用者固定打同一台後端，保住實例記憶體裡的 session。健康檢查失敗時會呼叫 `Remove`。它能編譯、單機測試會過。請做 code review，找出會在線上出問題的地方（至少兩個）。

```go title="proxy.go"
func (p *Proxy) pick(r *http.Request) *url.URL {
	h := fnv.New32a()
	h.Write([]byte(r.RemoteAddr))

	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.backends) == 0 {
		return nil
	}
	return p.backends[h.Sum32()%uint32(len(p.backends))]
}

// Remove drops a backend that failed its health check.
func (p *Proxy) Remove(dead *url.URL) {
	p.mu.Lock()
	defer p.mu.Unlock()
	kept := p.backends[:0]
	for _, b := range p.backends {
		if b.String() != dead.String() {
			kept = append(kept, b)
		}
	}
	p.backends = kept
}
```

完整檔案在 `recognize/proxy.go`，`go test ./...` 不會掃到它。

<details>
<summary>參考解答</summary>

1. **`pick` 拿 `r.RemoteAddr` 當 key**：`RemoteAddr` 的格式是 `203.0.113.7:51724`，後面的 port 是客戶端每條新連線隨機配的。同一個使用者換一條連線，key 就變了，被導向別台，session 保持形同虛設，單機測試用同一條連線所以測不出來。正確寫法：用 `net.SplitHostPort` 去掉 port，更好的是用 session cookie 之類的使用者識別碼當 key，並意識到 IP 會受 NAT 影響。
2. **`% len(p.backends)` 加上 `Remove` 縮短 slice**：任何一台被健康檢查剔除，剩下的台數變了，大部分 key 的餘數跟著變，3 台掉 1 台時約三分之二的使用者換台；恢復後又換一次。正確寫法：用一致性雜湊（如範例的 `balancer.Ring`），或把掛掉的後端標成 `down` 而不從陣列刪除，讓其他 key 的對應保持不變，只把落在掛掉那台的 key 改派。

</details>

## 快閃卡

<details>
<summary>nginx 的 upstream 是什麼？</summary>

一組後端 server 的群組，在 <code>upstream</code> 區塊裡列出，再用 <code>proxy_pass</code> 指向它。選哪一台由這個群組的演算法決定，預設是加權輪詢。

</details>

<details>
<summary>L4 的 stream 與 L7 的 http 負載平衡，差在哪？</summary>

L7 解析 HTTP，每個請求各選一次，能依 URI、header、cookie 決定去向；L4 只轉發 TCP 或 UDP，每條連線只在建立時選一次。

</details>

<details>
<summary>三台後端 weight 為 5、1、1，為什麼不是連續 5 次給 a？</summary>

nginx 用平滑加權輪詢：每次把 weight 加到分數、挑最高分、再從贏家扣掉總 weight，所以 a 的 5 次被 b、c 的回合隔開，順序是 a a b a c a a。

</details>

<details>
<summary>請求耗時從 5 ms 到 3 秒都有，該用輪詢還是 least_conn？</summary>

用 <code>least_conn</code>。輪詢不看實際負載，慢請求會碰巧堆在某一台；<code>least_conn</code> 選處理中連線數除以 weight 最小的那台，會避開正在忙的後端。

</details>

<details>
<summary>三台 nginx 同時用 least_conn 選後端，可能發生什麼問題？</summary>

群聚效應：大家看到的「最空的後端」是同一台，同時衝向它，把它瞬間灌爆。<code>random two least_conn</code> 先隨機挑兩台再比較，能分散各個 nginx 的選擇。

</details>

<details>
<summary>1,000 位員工從同一個公司出口 203.0.113.0/24 連進來，使用 ip_hash，會分到幾台後端？</summary>

只會分到 1 台。<code>ip_hash</code> 用 IPv4 的前三個位元組當 key，同一個 /24 的所有人雜湊相同，全部落在同一台。

</details>

<details>
<summary>hash 加上 consistent 的好處是什麼？</summary>

增減後端時只有少數 key 換台。不加時 key 的雜湊對 server 數量取餘數，3 變 4 台約 75% 的 key 換台；加上後約 25%，所以 session 與本機快取多半保得住。

</details>

<details>
<summary>用 hash $cookie_sid consistent 後，登入頁的流量全打到同一台，為什麼？</summary>

登入頁的請求還沒有 cookie，<code>$cookie_sid</code> 是空字串，雜湊固定，全部指向同一個點。只在需要保持 session 的路徑使用這個 upstream。

</details>

<details>
<summary>least_conn 的結果和輪詢差不多，慢的後端仍然堆積連線，先檢查什麼？</summary>

檢查 upstream 有沒有設 <code>zone</code>。沒設時，每個 worker 各自維護自己的狀態，只看到自己經手的連線；有了 <code>zone</code> 才在 worker 間共享。

</details>

## 延伸閱讀

- [Module ngx_http_upstream_module](https://nginx.org/en/docs/http/ngx_http_upstream_module.html)：upstream、`least_conn`、`hash`、`random`、`zone` 的完整參數與說明
- [HTTP Load Balancing](https://nginx.org/en/docs/http/load_balancing.html)：官方的負載平衡入門，含各種方法的示範設定
- [Module ngx_stream_upstream_module](https://nginx.org/en/docs/stream/ngx_stream_upstream_module.html)：L4 的 upstream 設定，與 http 版本的差異
- [NGINX and the "Power of Two Choices" Load-Balancing Algorithm](https://www.nginx.com/blog/nginx-power-of-two-choices-load-balancing-algorithm/)：為什麼 `random two` 在多個負載平衡器並存時更穩
- [The Power of Two Choices in Randomized Load Balancing](https://www.eecs.harvard.edu/~michaelm/postscripts/tpds2001.pdf)：Mitzenmacher 的論文，說明隨機挑兩個再比較為什麼有效
