---
title: Container 與 Image 的攻擊面
description: 在本機重現殘留機密與 docker.sock 濫用，再用掃描、固定 digest 與最小權限擋下它們。
sidebar:
  order: 1
---

## 先想想

你把短網址服務 v1 打包成映像（image），Dockerfile 長這樣：

```dockerfile title="Dockerfile.leaky"
FROM alpine:3.20
WORKDIR /app
COPY demo.env /app/.env
RUN rm /app/.env
CMD ["sh"]
```

`demo.env` 裡有管理用的 `ADMIN_TOKEN`。同事 code review 時發現 token 不該留在映像裡，所以加了最後的 `RUN rm /app/.env`，然後把映像推到公開的 registry。

陌生人 `docker pull` 之後，拿得到 `ADMIN_TOKEN` 嗎？如果拿得到，它藏在映像的哪裡？

<details>
<summary>提示</summary>

想想 Dockerfile 的每一行指令各自產生什麼，以及 `rm` 這個指令到底是「刪掉檔案」，還是「記下檔案被刪掉」。

</details>

## 是什麼

**容器與映像的攻擊面（container and image attack surface）** 是攻擊者在兩個階段能利用的錯誤設定與信任假設總和：映像被建出來、流通時，以及容器被啟動、執行時。

映像是一疊唯讀的檔案系統 layer 加上啟動設定；容器是用映像啟動的行程，靠 namespace、cgroup 與 capabilities 隔離。它和虛擬機最大的差別是：所有容器與主機共用同一個核心。所以容器的邊界是「設定出來的」，每多給一個權限，邊界就薄一層。

## 為什麼存在

映像是別人的程式加上你的設定，而且人們習慣直接信任它。兩項公開研究量化了後果：

- 亞琛工業大學（RWTH Aachen）的研究團隊分析了 Docker Hub 上 337,171 個映像與另外 8,076 個私有 registry 的映像，約 8.5% 含有機密，外洩的內容包括 52,107 把私鑰與 3,158 個 API 金鑰。
- Sysdig 在 2022 年掃描約 250,000 個 Docker Hub 上未經驗證的 Linux 映像，標出 1,652 個惡意映像，最多的是挖礦程式。它們常取名成熱門開源軟體的樣子，等人誤拉。

執行期的問題則是 Docker 把方便放在安全前面。預設容器以 root 執行、預設帶 14 項 capabilities、tag 可以被改指到別的映像。每一個「為了讓它跑起來」加上的旗標（`--privileged`、掛 `docker.sock`、`--cap-add`），都在拆容器和主機之間的牆。

攻擊方要學這些，是因為它們在滲透測試裡是常見的第一站：拿到一個容器的 shell 之後，先看這個容器被給了多少權限，往往比找核心漏洞快得多。

## 怎麼運作

:::caution
以下實驗只在你自己機器上的 Docker 進行，只建立本機容器與映像。`docker.sock` 的實驗會讓容器掛載你的 Docker 主機的根目錄，請不要在共用機器或正式環境執行。不要拿這些技巧對你沒有書面授權的系統測試。
:::

範例在 `playground/cyber/p4-advanced/4-3-cloud-container/container-image-attack-surface/`，以下指令都在該目錄執行。它是一個小工具 `containeraudit`，三個子命令對應三個位置：`image` 掃映像的 layer、`dockerfile` 檢查 Dockerfile、`run` 稽核 `docker inspect` 的輸出。每個子命令有發現就以 exit code 1 結束，可以直接接在 CI 上擋住建置。

### 攻擊面地圖

```text
Dockerfile ──build──► 映像（多個 layer）──push──► registry ──pull──► 主機上的容器
    │                      │                        │                    │
 ① 把機密 COPY 進去    ② 基底映像或 tag         ③ 拉到別人做的         ④ 啟動旗標給太多權限
    再刪掉              會變                       惡意映像              （docker.sock、
                                                                          privileged、caps）
```

看圖時要分清楚攻擊者站在哪：①②③ 發生在容器還沒跑起來之前，攻擊者是「映像的提供者或取得者」；④ 發生在容器跑起來之後，攻擊者已經在容器裡，問題是「這個容器能做什麼」。

### 供應鏈一：殘留機密

Dockerfile 的每一個 `COPY`、`RUN` 都會產生一個新的 layer，layer 一旦建好就不再改動。`RUN rm /app/.env` 不會回頭改前一個 layer，它只是新增一個 layer，裡面放一個叫 whiteout 的標記（檔名加上 `.wh.` 前綴），告訴檔案系統「這個檔案在這一層之後視為不存在」。

我們把 Dockerfile 建起來，存成 tar，用工具掃每個 layer：

```bash
docker build -f lab/Dockerfile.leaky -t sho-leaky:1 lab
docker save sho-leaky:1 -o leaky.tar
go run . image leaky.tar
```

```text
layer 3  /app/.env  [secret-file, secret-assignment]  deleted in layer 4, but layer 3 still stores it
```

工具能看見它，是因為它逐個 layer 處理，而不是只看最終的檔案系統。核心是這段，它依序重播每個 layer：whiteout 只會標記比它更早的 layer 裡的檔案：

```go title="imagescan.go"
func merge(results []layerResult) []Finding {
	var out []Finding
	for i, res := range results {
		for _, w := range res.whiteouts {
			for j := range out {
				if out[j].DeletedIn == 0 && covers(w, out[j].Path) {
					out[j].DeletedIn = i + 1
				}
			}
		}
		for _, f := range res.findings {
			f.Layer = i + 1
			out = append(out, f)
		}
	}
	return out
}
```

攻擊者連工具都不需要，`tar` 就夠了：

```bash
mkdir leaked && tar xf leaky.tar -C leaked
for b in leaked/blobs/sha256/*; do tar tf "$b" 2>/dev/null | grep -q '^app/.env$' && tar xOf "$b" app/.env; done
```

```text
ADMIN_TOKEN=sho-demo-0123456789abcdef
```

`docker save` 輸出的每個 layer 是 `blobs/sha256/` 下的一個 tar。第一個 `tar` 解開外層，迴圈再逐個打開 layer，只要有 `app/.env` 就印出內容。`docker pull` 拉下來的 layer 內容一樣，所以容器裡看不到檔案，不代表映像裡沒有。

### 供應鏈二：基底映像與 tag

`FROM alpine:3.20` 裡的 `3.20` 是 tag，一個可以被改指的標籤，不是內容的指紋。同一個 tag 今天指向映像 A，明天可以被重新推送成映像 B，你的下一次建置就悄悄換了基底。用本機重現這個行為：

```bash
docker tag alpine:3.20 sho-base:1.0
docker image inspect sho-base:1.0 --format '{{.Id}}'
docker tag sho-leaky:1 sho-base:1.0
docker image inspect sho-base:1.0 --format '{{.Id}}'
```

```text
sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc
sha256:8c5b887a69b57b5494231d91b7d6d663a0a866bbb6f796d08196f88c8c7ccaec
```

名字 `sho-base:1.0` 沒變，指向的映像卻不同。公開 registry 上，帳號被盜或維護者換人，就是同一件事發生在別人的 tag 上。

**digest** 才是內容的雜湊值，同樣的 digest 一定是同樣的內容。寫成 `alpine:3.20@sha256:d9e853…` 就把基底固定了。`dockerfile` 子命令的第一條規則就檢查這件事：

```go title="dockerlint.go"
if image != "scratch" && !stages[strings.ToLower(image)] && !strings.Contains(image, "$") && !strings.Contains(image, "@sha256:") {
	add("unpinned-base", "tag can move to a different image; pin "+image+" by @sha256 digest")
}
```

惡意映像是同一個信任問題的另一端：任何人都能在公開 registry 發布映像，名字可以取得像熱門軟體。`docker run` 之後，映像的 `ENTRYPOINT` 就以你給的權限執行，它可以挖礦、可以讀你掛進去的任何檔案。你給它的權限愈大，它能做的愈多，這就把我們帶到執行期。

### 執行期一：docker.sock

`docker` 命令列其實只是一個客戶端，它透過 Unix socket `/var/run/docker.sock` 對 Docker daemon 下指令，而 daemon 以 root 執行。誰能連上這個 socket，誰就能要求 daemon 啟動任何容器，包括掛載主機根目錄的容器。很多監控、CI 工具為了「看到其他容器」把這個 socket 掛進容器，等於給了容器主機的 root。

```bash
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock docker:cli sh -c '
  echo "outer hostname: $(hostname)"
  docker run --rm -v /:/host alpine:3.20 sh -c "echo inner sees host hostname: \$(cat /host/etc/hostname)"'
```

```text
outer hostname: f95795caac0e
inner sees host hostname: orbstack
```

外層容器的 hostname 是容器 ID。它在裡面再開了一個容器，把 Docker 主機的根目錄掛在 `/host`，讀到的卻是主機自己的 hostname。（輸出依你的機器而異；這裡是在 OrbStack 上跑出來的。）外層容器沒有任何特權旗標，它只有那個 socket。

常見的誤解是把 socket 掛成唯讀就安全了。實測：

```bash
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock:ro docker:cli docker ps
```

照樣列出所有容器。`:ro` 只讓檔案系統不能被寫入，而 `connect()` 到一個 socket 並不會寫那個檔案，所以對 socket 完全沒有限制效果。OWASP 的 Docker 速查表也明確說掛成唯讀不是解法，只會讓利用稍微難一點。稽核工具因此不管 mount 是讀寫還是唯讀，一律回報：

```go title="runaudit.go"
for _, m := range c.Mounts {
	// A read-only mount (:ro) does not help: connect() never writes to the socket file.
	if path.Base(m.Source) == "docker.sock" {
		add("critical", "docker-sock", "anyone inside can ask the daemon to start a container that mounts the host: "+m.Source)
	} else if slices.Contains(sensitiveHostPaths, path.Clean(m.Source)) {
		add("high", "sensitive-host-path", "host directory mounted into the container: "+m.Source)
	}
}
```

### 執行期二：privileged 與 capabilities

Linux 把 root 的權力拆成大約四十項 **capabilities**，例如 `NET_ADMIN`（改網路設定）、`SYS_ADMIN`（mount、namespace 操作，近乎 root）。行程所擁有的集合寫在 `/proc/<pid>/status` 的 `CapEff`。我們比較四種啟動方式：

```bash
docker run --rm alpine:3.20 sh -c 'grep CapEff /proc/1/status; ip link add dummy0 type dummy; ls /dev | wc -l'
docker run --rm --cap-drop ALL alpine:3.20 sh -c 'grep CapEff /proc/1/status'
docker run --rm --cap-add SYS_ADMIN alpine:3.20 sh -c 'grep CapEff /proc/1/status'
docker run --rm --privileged alpine:3.20 sh -c 'grep CapEff /proc/1/status; ip link add dummy0 type dummy && echo link-added; ls /dev | wc -l'
```

| 啟動方式 | `CapEff` | `ip link add` | `/dev` 項目數 |
| --- | --- | --- | --- |
| 預設 | `00000000a80425fb`（14 項） | `Operation not permitted` | 14 |
| `--cap-drop ALL` | `0000000000000000` | `Operation not permitted`（見下方加固實測） | 本文未測 |
| `--cap-add SYS_ADMIN` | `00000000a82425fb` | 本文未測 | 本文未測 |
| `--privileged` | `000001ffffffffff`（全部） | 成功（`link-added`） | 51 |

預設的 14 項已經比最小需求多，所以防守要從 `--cap-drop ALL` 開始，再只加回需要的。`--privileged` 則一次給齊全部 capabilities，還把主機的裝置都掛進容器，並關掉 seccomp 與 AppArmor 的預設限制。`--cap-add SYS_ADMIN` 只多了一個位元，但那一項本身就接近 root。

拿到這些權限之後怎麼跨過容器與主機的邊界（例如掛載主機磁碟、濫用 cgroup），是下一篇「容器逃逸」的主題。本篇到「權限被給了多少」為止。稽核工具把這些分成幾條規則：

```go title="runaudit.go"
var dangerousCaps = map[string]string{
	"SYS_ADMIN":       "mount, namespace and cgroup control: close to root",
	"SYS_PTRACE":      "attach to other processes, including host processes under --pid=host",
	"SYS_MODULE":      "load kernel modules",
	"DAC_READ_SEARCH": "bypass file read permission checks",
	"NET_ADMIN":       "change interfaces, routes and firewall rules",
}
```

對一個把 `--privileged`、`docker.sock`、`--pid host` 全用上的日誌代理容器（輸入是真實 `docker inspect` 輸出整理成的 `testdata/inspect-privileged.json`）：

```bash
go run . run testdata/inspect-privileged.json
```

```text
critical sho-log-agent  privileged  all capabilities, all host devices, no seccomp or AppArmor profile
critical sho-log-agent  docker-sock  anyone inside can ask the daemon to start a container that mounts the host: /var/run/docker.sock
high     sho-log-agent  host-pid  the container sees and can signal every host process
medium   sho-log-agent  new-privileges-allowed  a setuid binary inside can still gain privileges
medium   sho-log-agent  runs-as-root  the main process runs as uid 0 inside the container
medium   sho-log-agent  writable-rootfs  an attacker can drop tools and persist inside the container filesystem
```

### 修補並驗證

殘留機密的修法是讓機密根本不進 layer。BuildKit 的 secret mount 只在單一 `RUN` 執行期間把檔案掛在 `/run/secrets/`，不寫進任何 layer；同時把基底固定到 digest，並切到非 root 使用者：

```dockerfile title="Dockerfile.fixed"
FROM alpine:3.20@sha256:d9e853e87e55526f6b2917df91a2115c36dd7c696a35be12163d44e6e2a4b6bc
WORKDIR /app
# The token exists only while this RUN step executes: it is mounted from
# the host at build time and never written into a layer.
RUN --mount=type=secret,id=admin_token test -s /run/secrets/admin_token
USER 65532:65532
CMD ["sh"]
```

驗證修補後攻擊確實失效：

```bash
docker build -f lab/Dockerfile.fixed --secret id=admin_token,src=lab/demo.env -t sho-fixed:1 lab
docker save sho-fixed:1 -o fixed.tar
go run . image fixed.tar
go run . dockerfile lab/Dockerfile.fixed
```

```text
ok: nothing found
ok: nothing found
```

layer 掃描沒有找到機密，`docker history --no-trunc` 也只看得到檔案「路徑」，看不到內容。執行期的修法是用 `--cap-drop ALL`、`--read-only`、非 root 使用者與 `no-new-privileges` 啟動，並且不掛 `docker.sock`：

```bash
docker run --rm --cap-drop ALL --read-only --user 65532:65532 --security-opt no-new-privileges alpine:3.20 \
  sh -c 'id -u; grep CapEff /proc/1/status; ip link add dummy0 type dummy'
```

```text
65532
CapEff:	0000000000000000
ip: RTNETLINK answers: Operation not permitted
```

以這組旗標啟動的容器，`docker inspect` 輸出丟給稽核工具會通過（`testdata/inspect-hardened.json`，`TestHardenedContainerIsClean` 驗證）。

| 攻擊 | 防禦 | 怎麼驗證 |
| --- | --- | --- |
| 殘留機密 | BuildKit secret mount；`.dockerignore` 排除 `.env`；機密在執行時才注入 | `containeraudit image` 無發現 |
| 基底或 tag 被換 | `FROM` 固定 `@sha256` digest；只用組織核准的基底 | `containeraudit dockerfile` 無 `unpinned-base` |
| 惡意映像 | 只從官方或 Verified Publisher 拉；簽章與掃描；先在無權限環境試跑 | 映像來源可追溯、掃描無已知惡意 |
| `docker.sock` | 不掛；必要時用只放行讀取 API 的 socket proxy | `containeraudit run` 無 `docker-sock` |
| `--privileged` | 不用；缺什麼就單獨 `--cap-add` 那一項 | 無 `privileged` |
| 過寬 capabilities | `--cap-drop ALL` 再加回；非 root；`no-new-privileges`；唯讀 rootfs | 無 `dangerous-capability`、`capabilities-not-dropped` |

回到一開始的問題：拿得到。`RUN rm /app/.env` 沒有刪除任何東西，它只在第 4 個 layer 放了一個 whiteout；第 3 個 layer（`COPY` 產生的那層）完整保留了 `.env`，任何人用 `docker save` 或 `docker pull` 取得映像，再逐層打開就讀得到 `ADMIN_TOKEN`。容器裡執行 `ls /app` 看不到它，只代表合併後的檔案系統隱藏了它。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 掃描 layer 與 Dockerfile（本篇工具） | 便宜，可放進 CI 擋建置 | 只認已知模式，認不出自訂格式的機密 | 每次建置的第一道檢查 |
| 固定 digest | 內容不會被悄悄換掉 | 不會自動拿到安全更新，需要更新流程 | 所有正式映像的基底 |
| BuildKit secret mount | 機密不進任何 layer | 建置需要 BuildKit，要改 Dockerfile | 建置時需要私有套件庫或金鑰 |
| `--cap-drop ALL` 加回清單 | 權限最小，攻擊面最小 | 服務啟動失敗時要查缺哪一項 | 預設做法 |
| Socket proxy | 只放行必要的 API | 多維護一個元件，設定出錯仍會過寬 | 工具真的需要讀容器清單 |
| 沙箱型 runtime（如 gVisor）或每個工作負載一台 microVM | 不共用核心，邊界比容器強 | 效能與相容性成本 | 執行不可信的程式碼、多租戶 |

什麼時候這些防禦不夠或不該照搬：

- **只靠掃描不夠。** 掃描器找得到 `.env` 與常見格式的金鑰，找不到你自己的格式，也找不到「合法但不該給這個容器的權限」。它是下限，不是保證。
- **`--cap-drop ALL` 要看服務。** 服務如果要綁 1024 以下的 port，就會缺 `NET_BIND_SERVICE`；與其加回這項，不如改聽 8080 之類的高 port，再由前面的負載平衡轉接。
- **容器加固不能取代隔離。** 如果你要跑的是不可信的程式碼（例如讓使用者上傳程式再執行），共用核心這件事本身就是風險，應該換沙箱型 runtime 或 microVM，而不是繼續加旗標。

## 常見坑與故障案例

### 以為 `rm` 或後續 layer 刪掉了機密

**症狀**：Dockerfile 先 `COPY .env`，建置完 `RUN rm`，容器裡找不到檔案，就當作乾淨了。

**原因**：如前所述，刪除只是新增 whiteout，舊 layer 原封不動。RWTH Aachen 的研究就是逐 layer 掃描上述映像，找出約 8.5% 含有機密，包括私鑰；這些金鑰不少還被實際在用（研究在網際網路上找到使用外洩私鑰的 TLS 與 SSH 主機）。

**對策**：機密不進 `COPY`；用 secret mount；`.dockerignore` 排除 `.env`；已經推出去的映像，視為機密外洩，要換掉機密，而不是只重建映像。

### 把機密放在 `ARG` 或 `ENV`

**症狀**：為了方便，寫 `ARG NPM_TOKEN` 並用 `--build-arg` 傳入，或寫 `ENV DB_PASSWORD=...`。

**原因**：這兩個都會存在映像的 metadata 裡，`docker history` 與 `docker inspect` 看得到，任何拿到映像的人都能讀。Docker 自己的建置檢查（`SecretsUsedInArgOrEnv`）也警告這是不安全的。`dockerfile` 子命令用 `secret-in-arg`／`secret-in-env` 規則抓這件事。

**對策**：建置時用 secret mount，執行時由編排系統注入。

### 信任 tag 與看起來像官方的名字

**症狀**：`FROM node:latest`，或在 Docker Hub 搜尋後直接用第一個名字相近的映像。

**原因**：`latest` 與任何 tag 都可能被改指；公開 registry 上任何人都能發布映像。Sysdig 2022 年的研究就是在 250,000 個未經驗證的映像裡找出 1,652 個惡意映像，其中很多取名成熱門開源軟體的樣子。

**對策**：基底固定 digest、只用官方或 Verified Publisher，並讓組織有一份核准的基底清單（DevOps 領域的 [Distroless 映像](/devops/l1/1-2-container-image/distroless-image/) 講怎麼縮小映像內容，Golden Image 一篇會講怎麼強制只能從核准的基底開始）。

### 為了「看到其他容器」掛上 docker.sock

**症狀**：監控、日誌或 CI 容器需要列出別的容器，於是掛 `/var/run/docker.sock`，還特別加上 `:ro` 以為這樣就安全。

**原因**：socket 沒有唯讀版本；能連上就能叫 daemon 開任何容器。（示意案例：本書沒有可靠的公開事故報告可引用，這是依 Docker daemon 的運作方式與前面的實測整理出的情境。）

**對策**：不掛。真的要讀容器清單，用只放行讀取 API 的 socket proxy，或改走日誌驅動，不要碰 daemon API。

### 遇到「Operation not permitted」就加 `--privileged`

**症狀**：服務在容器裡啟動失敗，錯誤是 `Operation not permitted`，Stack Overflow 的第一個答案是 `--privileged`，加了就好。

**原因**：預設只有 14 項 capabilities，缺的那一項多半只有一項；`--privileged` 卻把全部 41 位元都打開，外加所有裝置。（示意案例：這是常見的除錯慣性，非特定事故。）

**對策**：從錯誤訊息與 `capabilities(7)` 說明找出缺的那一項 capability，只 `--cap-add` 它；加完後把設定交給 `containeraudit run` 驗一次。

## 小練習

範例在 `playground/cyber/p4-advanced/4-3-cloud-container/container-image-attack-surface/`。先 `go test ./...` 確認全部通過。

**1. 證明 `:ro` 擋不住。** 在 `runaudit/runaudit_test.go` 加一個測試：一個 `docker.sock` 的 mount 帶 `"RW": false`，其餘設定都已加固（`CapDrop` 為 `ALL`、非 root、唯讀 rootfs、`no-new-privileges`），預期 `Audit` 仍然只回報 `docker-sock`。這個測試能通過，說明了什麼？

<details>
<summary>參考解答</summary>

```go
func TestReadOnlyDockerSockIsStillCritical(t *testing.T) {
	const sample = `[{"Name":"/x","Config":{"User":"65532"},"HostConfig":{
		"CapDrop":["ALL"],"ReadonlyRootfs":true,"SecurityOpt":["no-new-privileges"]},
		"Mounts":[{"Source":"/var/run/docker.sock","Destination":"/var/run/docker.sock","RW":false}]}]`
	if got, want := rules(t, sample), []string{"docker-sock"}; !slices.Equal(got, want) {
		t.Errorf("rules = %v, want %v", got, want)
	}
}
```

測試通過代表稽核規則完全不看 `RW`：唯讀與否都一樣危險。因為 `:ro` 限制的是對檔案的寫入，而連到 socket 不會寫那個檔案；你也可以用 `docker run -v /var/run/docker.sock:/var/run/docker.sock:ro docker:cli docker ps` 實測，照樣列得出容器。

</details>

**2. 修好 `Dockerfile.leaky`。** 不看 `Dockerfile.fixed`，自己改 `lab/Dockerfile.leaky`，直到 `go run . dockerfile lab/Dockerfile.leaky` 回報 `ok: nothing found`（exit code 0），並且建出來的映像用 `go run . image` 掃不到機密。

<details>
<summary>參考解答</summary>

`lab/Dockerfile.fixed` 就是一份答案：基底加 `@sha256:` digest、用 `RUN --mount=type=secret` 取代 `COPY` 加 `rm`、最後加 `USER 65532:65532`。三個發現各自對應一個修法：`unpinned-base`、`secret-copied`、`runs-as-root`。建置時要加 `--secret id=admin_token,src=lab/demo.env`，不加的話 `test -s` 會失敗，建置中止。

</details>

### 辨識

下面這份 `compose.yaml` 由 AI 產生，目的是「跑短網址服務，並讓 log-agent 把服務的容器日誌送去儀表板」。它能啟動，儀表板也能收到日誌。請做 code review，找出它的問題（至少兩個）。

```yaml title="compose.yaml"
services:
  shortener:
    image: sho.rt/shortener:latest
    ports:
      - "8080:8080"
    environment:
      ADMIN_TOKEN: ${ADMIN_TOKEN}
    user: "65532:65532"
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true

  log-agent:
    image: sho.rt/log-agent@sha256:5c3f1d0e8a7b4c2d9e6f1a3b5c7d9e0f2a4b6c8d0e1f3a5b7c9d1e3f5a7b9c0d
    volumes:
      # follow the shortener's container logs for the dashboard
      - /var/run/docker.sock:/var/run/docker.sock:ro
    privileged: true
    restart: unless-stopped
```

完整檔案在 `recognize/compose.yaml`，標明是 AI 產生，不會被 `go test ./...` 掃到。

<details>
<summary>參考解答</summary>

1. **`/var/run/docker.sock:/var/run/docker.sock:ro`**：`:ro` 不會阻止連到 socket 並呼叫 daemon API，log-agent 被入侵後可以要求 daemon 開一個掛載主機根目錄的容器，等於主機 root。正確寫法是移除這行；要日誌就改用日誌驅動，或只放行讀取 API 的 socket proxy。
2. **`privileged: true`**：log-agent 讀日誌不需要任何特權，這行給了它全部 capabilities 與所有主機裝置，並關掉 seccomp 與 AppArmor。正確寫法是整行移除，改成 `cap_drop: [ALL]`、非 root 使用者、`read_only: true`、`no-new-privileges`，與 `shortener` 一致。
3. **`image: sho.rt/shortener:latest`**：`latest` 是會移動的 tag，下次部署可能拉到不同的映像；`log-agent` 有固定 digest，`shortener` 卻沒有，標準不一致。正確寫法是改成 `sho.rt/shortener@sha256:<digest>`，由 CI 在建置時寫入。

好的對照是 `shortener` 那一段：非 root、`cap_drop: ALL`、唯讀 rootfs、`no-new-privileges`，這就是 `testdata/inspect-hardened.json` 通過稽核的那組旗標。

</details>

## 快閃卡

<details>
<summary>為什麼 <code>RUN rm /app/.env</code> 刪不掉已經 <code>COPY</code> 進映像的檔案？</summary>

layer 一旦建好就不再改動。`rm` 只會在新的 layer 加一個 whiteout 標記，讓合併後的檔案系統看不到它；放著檔案的舊 layer 仍然完整留在映像裡。

</details>

<details>
<summary>你在容器裡 <code>ls /app</code> 看不到 <code>.env</code>，能說映像裡沒有這個檔案嗎？</summary>

不能。容器看到的是合併後的檔案系統，被 whiteout 的檔案會被隱藏。用 `docker save` 逐個打開 layer 仍然讀得到。

</details>

<details>
<summary>tag 和 digest 的差別是什麼？為什麼固定 digest 更安全？</summary>

tag 是可以被改指的標籤，同一個 tag 不同時間可能指向不同映像；digest 是內容的雜湊值，內容不同 digest 就不同。固定 digest 就固定了你拿到的內容。

</details>

<details>
<summary>Dockerfile 寫了 <code>ARG NPM_TOKEN</code>，用 <code>--build-arg NPM_TOKEN=xxx</code> 建置。拿到映像的人看得到 token 嗎？</summary>

看得到。ARG 與 ENV 會存在映像的 metadata，`docker history` 與 `docker inspect` 都能讀。建置時要改用 secret mount，它不會寫進任何 layer。

</details>

<details>
<summary>容器 A 掛了 <code>-v /var/run/docker.sock:/var/run/docker.sock:ro</code>，A 被入侵後能在主機上開新容器嗎？</summary>

能。`:ro` 只限制寫入檔案，連到 socket 並不會寫那個檔案，所以仍可呼叫 Docker API，要求 daemon 啟動掛載主機根目錄的容器。

</details>

<details>
<summary>預設的容器有幾項 capabilities？<code>--privileged</code> 和 <code>--cap-add SYS_ADMIN</code> 差在哪？</summary>

預設 14 項。`--cap-add SYS_ADMIN` 只多一項（但那一項近乎 root）；`--privileged` 則給全部 capabilities，還掛載所有主機裝置並關掉 seccomp 與 AppArmor 的預設限制。

</details>

<details>
<summary>服務在容器裡報 <code>Operation not permitted</code>，為什麼不該直接加 <code>--privileged</code>？</summary>

預設只缺少一兩項 capability，`--privileged` 卻把全部都打開並放行所有裝置，權限遠超過需要。應找出缺的那一項，只 `--cap-add` 它。

</details>

<details>
<summary>一個容器用 <code>--cap-drop ALL</code> 啟動，在裡面執行 <code>ip link add dummy0 type dummy</code> 會成功嗎？</summary>

不會。`CapEff` 是全零，沒有 `NET_ADMIN`，會得到 `Operation not permitted`。預設的 14 項也不含 `NET_ADMIN`，同樣失敗。

</details>

<details>
<summary>容器加固得再好，為什麼不適合拿來跑完全不可信的使用者程式？</summary>

因為容器與主機共用同一個核心，邊界是設定出來的，不是硬體隔離。跑不可信程式應改用沙箱型 runtime 或 microVM。

</details>

## 延伸閱讀

- [Secrets Revealed in Container Images: An Internet-wide Study on Occurrence and Impact（Dahlmanns 等人，ACM ASIA CCS 2023）](https://arxiv.org/abs/2307.03958)：本篇引用的 Docker Hub 機密外洩數字的原始論文。
- [Analysis of supply chain attacks through public Docker images（Sysdig）](https://sysdig.com/blog/analysis-of-supply-chain-attacks-through-public-docker-images/)：1,652 個惡意映像的分析，含挖礦與偽裝名稱的手法。
- [Docker Security Cheat Sheet（OWASP）](https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html)：不掛 `docker.sock`、限制 capabilities 等規則的條列整理。
- [SecretsUsedInArgOrEnv（Docker Docs）](https://docs.docker.com/reference/build-checks/secrets-used-in-arg-or-env/)：官方說明為什麼 `ARG`／`ENV` 不該放機密，以及 secret mount 的寫法。
- [capabilities(7)（Linux man-pages）](https://man7.org/linux/man-pages/man7/capabilities.7.html)：每一項 capability 到底允許什麼的權威說明。
