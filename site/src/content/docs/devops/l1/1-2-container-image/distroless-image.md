---
title: Distroless 映像：去掉 shell 與套件管理器以縮小攻擊面，及其除錯取捨
description: 用 multi-stage build 把 Go 服務放進 distroless 映像，並知道沒有 shell 之後怎麼除錯。
sidebar:
  order: 3
---

## 先想想

短網址服務 v1 的 Dockerfile 只有三行：`FROM golang:1.26`、`COPY`、`go build`。映像推上倉庫後，CI 的掃描器報出一長串漏洞，其中不少出在 `curl`、`apt`、`perl` 這些套件上，而你的服務只是一個 Go 執行檔，從來沒呼叫過它們。

另一個情境：假設服務有個 bug，讓攻擊者能在容器裡執行任意指令。

這個映像裡的 `curl`、`apt` 和 Go 編譯器，對那位攻擊者有什麼用？如果映像裡只剩你的執行檔，攻擊者拿到同樣的 bug 之後，還剩下什麼可做？

<details>
<summary>提示</summary>

想想攻擊者拿到「能執行指令」的能力之後，下一步通常要下載工具、開一個 shell、翻找檔案。這些動作各自需要映像裡有什麼？

</details>

## 是什麼

**Distroless 映像** 是只包含你的應用程式與它的執行期相依，不含套件管理器、shell 和其他一般 Linux 發行版工具的容器映像。

「distroless」是「沒有發行版」的意思，指的是 Google 的開源專案 [GoogleContainerTools/distroless](https://github.com/GoogleContainerTools/distroless)。它的基底仍然來自 Debian 的套件，但只挑出必要的檔案，例如 CA 憑證、時區資料，需要時再加上 libc。它和 `scratch`（完全空白的映像）之間的差別是：distroless 替你備好了這些常用的檔案。

兩種映像裡的東西對照如下：

```text
┌────────────────────────────────────┬────────────────────────────────────┐
│ 一般基底（如 golang、debian）      │ Distroless（static）               │
├────────────────────────────────────┼────────────────────────────────────┤
│ 作業系統層                         │ 作業系統層                         │
│   shell（sh、bash）                │   （沒有 shell）                   │
│   套件管理器（apt、dpkg）          │   （沒有套件管理器）               │
│   命令列工具（ls、cat、curl…）     │   （沒有命令列工具）               │
│   libc、OpenSSL 等函式庫           │   CA 憑證、tzdata、/etc/passwd     │
│ 語言層（視映像而定）               │ 語言層                             │
│   編譯器、直譯器、標頭檔           │   （沒有編譯器）                   │
│ 你的應用程式                       │ 你的應用程式                       │
│   /shortener 執行檔                │   /shortener 執行檔                │
└────────────────────────────────────┴────────────────────────────────────┘
```

左右兩欄看「作業系統層」那幾行：左邊每一項都是攻擊者可以拿來用、掃描器會逐一比對版本的東西，右邊只剩執行檔需要的檔案。實際內容會隨映像與版本不同，這裡以 `static` 為例，它連 libc 也沒有。

## 為什麼存在

多數人從 `ubuntu`、`debian` 或語言官方映像（`golang`、`node`）開始。這些映像是為「開發時什麼都能用」設計的，裡面有套件管理器、shell 和一大堆命令列工具。在正式環境裡，這些東西有兩個代價。

**攻擊面變大。** 攻擊者靠程式漏洞取得指令執行能力之後，下一步是橫向擴張：用 `sh` 開互動式 shell，用 `curl` 或 `wget` 下載後續工具，用 `apt` 裝需要的套件。映像裡有這些，攻擊者就不必自帶。映像裡沒有，他每一步都得想別的辦法。

**漏洞掃描的雜訊變多。** 掃描器會逐一比對映像內每個套件的版本。套件越多，報出的 CVE 越多，而其中大多數你的服務根本沒用到。團隊看到幾百筆警告，就開始忽略它，真正該處理的那幾筆也跟著被淹沒。

Distroless 的做法是反過來：不是把不需要的東西一個個刪掉，而是從只有必要檔案的基底開始。我們在 arm64 的 Docker 上量過短網址服務的幾種打包方式：

| 打包方式 | 映像大小 |
| --- | --- |
| 單階段，直接用 `golang:1.26` | 1.41 GB |
| multi-stage，正式階段用 `debian:13-slim` | 147 MB |
| multi-stage，正式階段用 `distroless/static-debian13` | 15.8 MB |

數字會隨平台與版本略有出入，重點是量級：執行檔本身約 8.5 MB，其餘幾乎都是基底。

## 怎麼運作

:::note
本篇的範例都在 `playground/golang/devops/l1/1-2-container-image/distroless-image/`，要執行需要 Docker。
:::

### 兩個階段：先建置，再只帶產物

核心手法是 multi-stage build，在同一個 Dockerfile 裡用兩個 `FROM`：

```text
Builder 階段                              Production 階段
golang:1.26                               distroless/static:nonroot
┌──────────────────────┐                  ┌──────────────────────┐
│ Go 編譯器、原始碼     │   COPY --from    │ CA 憑證、tzdata       │
│ go build             │ ───────────────► │ /shortener 執行檔     │
│ → /out/shortener     │   只帶這一個檔    │ 沒有 shell、沒有 apt   │
└──────────────────────┘                  └──────────────────────┘
   用完即丟，不會被推上倉庫                    這才是你推上倉庫的映像
```

看這張圖要注意：Builder 階段的所有東西，包含編譯器與原始碼，都不會進入最終映像。只有 `COPY --from=build` 明確指定的檔案會被帶過去。

```dockerfile title="Dockerfile"
# Builder: full toolchain, never shipped.
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
# CGO off: a static binary has no libc dependency, which `static` does not have either.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/shortener .

# Production: no shell, no package manager, runs as UID 65532.
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/shortener /shortener
EXPOSE 8080
# Exec form is mandatory: there is no /bin/sh to interpret a shell-form command.
ENTRYPOINT ["/shortener"]
HEALTHCHECK --interval=10s --timeout=3s CMD ["/shortener", "-healthcheck"]
```

這個 Dockerfile 有三個細節，後面的「常見坑」會一一說明：`CGO_ENABLED=0`、`ENTRYPOINT` 用 JSON 陣列的寫法，以及 `HEALTHCHECK` 呼叫執行檔自己，而不是 `curl`。

### 選哪一個基底

專案提供多個映像家族，依你的執行檔需要什麼來選：

| 映像 | 內容 | 適用 |
| --- | --- | --- |
| `static` | CA 憑證、tzdata、`/etc/passwd`，沒有 libc | Go、Rust 等靜態連結的執行檔 |
| `base` | `static` 加上 glibc 與 OpenSSL | 動態連結 libc 的執行檔 |
| `cc` | `base` 加上 libgcc、libstdc++ | C++ 或需要這些函式庫的程式 |
| `java`、`nodejs`、`python3` | 對應的語言執行環境 | 用該語言寫的服務 |

Java 與 Node.js 的映像名稱帶版本，例如 `java21-debian13`、`nodejs24-debian13`，可用的版本以專案 README 為準。每個映像都有四種 tag：`latest`、`nonroot`、`debug`、`debug-nonroot`。正式環境用 `nonroot`，`debug` 系列多了一個 busybox shell，只在除錯時用。

我們實測 `static-debian13:nonroot` 約 7.3 MB，`base-debian13:nonroot` 約 49.6 MB。能用 `static` 就用 `static`。

### 為什麼 `nonroot` 很重要

多數映像預設以 root（UID 0）執行，Dockerfile 沒寫 `USER` 就是如此。容器內的 root 和 host 上的 root 是同一個 UID，只是被 namespace、capabilities 與 seccomp 圈起來。一旦容器逃逸的漏洞出現，攻擊者直接就是 host 上的 root。

`:nonroot` tag 讓程序以 UID 65532 執行。就算攻擊者拿到了指令執行能力，也寫不進系統目錄、改不了其他程序的檔案。Kubernetes 的 Pod Security Standards 在 `restricted` 等級也要求容器不能以 root 執行。

這是兩層獨立的防護：沒有 shell 讓攻擊者少了工具，不是 root 讓攻擊者少了權限。

### 沒有 shell，怎麼辦

驗證一下。同一個服務打包成單階段與 distroless 兩種映像，各自嘗試開 shell：

```bash
docker run --rm --entrypoint sh sho-single -c 'id; which curl apt-get'
docker run --rm --entrypoint sh sho-distroless -c 'id'
```

```text
uid=65532 gid=65532 groups=65532
/usr/bin/curl
/usr/bin/apt-get
...
exec: "sh": executable file not found in $PATH
```

第一個映像（一般的 `golang` 基底）有 shell、`curl` 和 `apt-get`；第二個什麼都找不到，連 `docker exec` 進去也一樣。

### Google 的作法：Bazel

distroless 專案自己用 Bazel 建置，Google 內部的服務也常用 Bazel 直接產出映像，不寫 Dockerfile。用 Bazel 的 [rules_oci](https://github.com/bazel-contrib/rules_oci) 可以宣告「以某個 distroless 映像為基底，加上這個執行檔」，建置結果可重現，也不需要 Docker daemon。對只有幾個服務的團隊，multi-stage Dockerfile 已經夠用；Bazel 在 monorepo、大量服務時才划算。

### 回答「先想想」

回到一開始的問題：`curl` 讓攻擊者下載後續工具，`apt` 讓他安裝任何需要的程式，編譯器讓他現場編譯攻擊程式碼，shell 讓他互動式地探索。Distroless 映像裡這些都沒有，攻擊者拿到同一個 bug 之後，只能呼叫你的執行檔本身提供的功能，或想辦法自己帶工具進來。這不代表不會被入侵，但每一步都變難了。掃描器要比對的套件也少了，雜訊跟著降低。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 完整發行版基底（`debian`、`ubuntu`） | 什麼工具都有，除錯最方便 | 體積大、CVE 多、攻擊面大 | 開發與測試環境 |
| Alpine | 體積小，有套件管理器 | 仍有 shell 與 `apk`；用 musl libc，與多數預編譯的二進位不相容 | 需要在映像裡裝東西的場景 |
| Distroless | 無 shell 與套件管理器，攻擊面小 | 除錯困難，要有嚴謹的建置流程 | 正式環境的服務 |
| `scratch` | 最小，什麼都沒有 | 連 CA 憑證和 tzdata 都要自己放 | 靜態執行檔且你清楚需要什麼 |
| Chainguard 等第三方硬化映像 | 同樣精簡，通常更新頻繁，提供 SBOM | 有商業授權或使用條件，要另外評估 | 組織有合規與供應鏈要求 |

Distroless 的好處在前面都講了：攻擊面小、CVE 少、體積小、映像內容固定而可預測。它的代價也要老實說：

- **除錯困難。** `kubectl exec` 進不去，因為沒有 shell。常見做法有兩種：換成 `:debug` tag 的映像（內含 busybox），或在 Kubernetes 用 `kubectl debug` 掛一個臨時的 debug 容器到同一個 Pod（要叢集支援 ephemeral container）。前者要重新部署，後者不必動原本的映像，後者通常是比較好的做法。
- **對直譯式語言沒那麼友善。** Go、Rust 編出一個靜態執行檔就完了。Python、Node.js 要把相依套件一起帶過去，帶原生擴充的套件還要和基底的 libc 與直譯器版本完全一致，不一致就在執行時才出錯。
- **日誌與監控要標準化。** 容器裡不能放 log agent 或監控 agent，也不能 `tail` 檔案。日誌必須寫 stdout 和 stderr，由容器執行環境收集；指標要由服務自己開 HTTP 端點。這本來就是好習慣，但現有服務可能得先改。
- **建置流程要更嚴謹。** 不能在最終映像裡 `RUN apt-get install`，因為沒有 `apt`。缺什麼都必須在 Builder 階段備好再複製過來。

什麼時候不該用：

- **服務還在頻繁排錯、團隊還不熟除錯手法時。** 先在測試環境用 `:debug` 熟悉流程，再推到正式環境。
- **服務必須在執行期安裝或呼叫外部程式時。** 例如在容器內執行 `ffmpeg`、`git` 或用 shell 腳本串流程。這種服務要嘛把工具一併放進映像，要嘛改架構，不要硬塞進 distroless。

## 常見坑與故障案例

### ENTRYPOINT 或 CMD 寫成 shell form

**症狀**：Dockerfile 寫 `ENTRYPOINT /shortener -addr :8080`，容器一啟動就結束。

**原因**：Shell form 會被包成 `/bin/sh -c "..."`，而 distroless 沒有 `/bin/sh`。

**怎麼避免**：一律用 exec form 的 JSON 陣列：`ENTRYPOINT ["/shortener", "-addr", ":8080"]`。需要環境變數展開時，改在程式裡讀環境變數。

### Go 執行檔是動態連結的

**症狀**：映像建置成功，容器一啟動就印出 `exec /shortener: no such file or directory`，但檔案明明就在。

**原因**：`golang` 映像裡有 gcc，`CGO_ENABLED` 預設為 1，標準函式庫的網路與使用者查詢會因此連結 libc。`static` 映像沒有 libc，核心找不到動態連結器，才報這個誤導人的錯誤。我們在範例中實測過。

**怎麼避免**：建置時加 `CGO_ENABLED=0`，或改用有 glibc 的 `base`。可以在 Builder 階段用 `file` 或 `ldd` 檢查執行檔是不是動態連結。

### HEALTHCHECK 還在呼叫 curl

**症狀**：容器跑得好好的，`docker ps` 卻永遠顯示 `unhealthy`。

**原因**：`HEALTHCHECK CMD curl -f ...` 是 shell form，需要 `/bin/sh` 與 `curl`，兩個都不存在。我們實測的 health log 是 `exec: "/bin/sh": stat /bin/sh: no such file or directory`。

**怎麼避免**：讓執行檔自己提供健康檢查，範例的 `-healthcheck` 旗標就是這個作法。在 Kubernetes 上更簡單，直接用 `httpGet` probe，讓 kubelet 去發 HTTP 請求，映像裡完全不必有檢查工具。

### 忘了 `nonroot`，容器內是 root

**症狀**：用 `distroless/static` 而不是 `distroless/static:nonroot`，一切正常，沒有任何警告。

**原因**：不帶 `nonroot` 的 tag 預設以 root 執行。映像看起來很精簡，其實少了一層保護。2019 年 runc 的 [CVE-2019-5736](https://nvd.nist.gov/vuln/detail/CVE-2019-5736) 就是容器逃逸漏洞，讓能在容器內以 root 身分執行指令的攻擊者覆寫 host 上的 runc 執行檔，進而取得 host 的 root 權限。

**怎麼避免**：基底選 `:nonroot`。多數情況下再讓 Kubernetes 的 `securityContext` 加上 `runAsNonRoot: true`，作為第二道檢查。

### 非 root 寫不了檔案

**症狀**：服務在本機用 root 跑得好好的，換成 `nonroot` 映像後，寫暫存檔或快取時報 `permission denied`。（示意案例）

**原因**：UID 65532 對根目錄與大多數系統目錄沒有寫入權限。

**怎麼避免**：把需要寫的路徑掛成 volume 或 `emptyDir`，或在 Builder 階段先建好目錄並用 `COPY --chown=65532:65532` 複製進去。

## 小練習

範例程式在 `playground/golang/devops/l1/1-2-container-image/distroless-image/`。先執行 `go test ./...` 確認全部通過。

**1. 比較兩種打包方式。** 目錄裡有 `Dockerfile.single`（單階段）與 `Dockerfile`（multi-stage + distroless）。分別建置成 `sho-single` 與 `sho-distroless`，用 `docker images` 比較大小，再各執行一次 `docker run --rm --entrypoint sh <映像> -c 'id'`。預測結果，再對照實際輸出。

<details>
<summary>參考解答</summary>

```bash
docker build -f Dockerfile.single -t sho-single .
docker build -t sho-distroless .
docker images | grep sho-
```

`sho-single` 約 1.4 GB，`sho-distroless` 約 16 MB（依平台略有不同）。第一個指令會印出 `uid=65532 ...`，因為單階段版本寫了 `USER 65532:65532`；第二個報 `executable file not found in $PATH`，因為映像裡沒有 `sh`。

`Dockerfile.single` 即使改成非 root，映像裡仍然有編譯器、`curl` 與 `apt`。兩者的差別不在 UID，而在攻擊者能用的東西。

</details>

**2. 證明健康檢查會回報失敗。** `health.go` 的 `probe` 在伺服器回非 200 時要回傳錯誤，因為容器平台是看結束碼判斷健康與否。在 `health_test.go` 加一個測試：伺服器回 500，`probe` 必須回傳錯誤。

<details>
<summary>參考解答</summary>

```go
func TestProbeServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if err := probe(srv.URL + "/healthz"); err == nil {
		t.Fatal("probe should fail when healthz returns 500")
	}
}
```

如果 `probe` 只檢查「請求有沒有送成功」而不看狀態碼，這個測試會失敗：伺服器壞了，但健康檢查永遠通過。

</details>

### 辨識

下面這份 Dockerfile 由 AI 產生，目的是「把短網址服務做成精簡又安全的 distroless 映像」。它能建置，容器啟動後 `curl localhost:8080/healthz` 回 200，服務運作正常。請做 code review，找出它的問題（至少兩個）。

```dockerfile title="Dockerfile"
FROM golang:1.26 AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -o /out/shortener .

FROM gcr.io/distroless/static-debian13
COPY --from=build /out/shortener /shortener
EXPOSE 8080
HEALTHCHECK CMD curl -f http://localhost:8080/healthz || exit 1
ENTRYPOINT ["/shortener"]
```

完整檔案在 `recognize/Dockerfile`，`go test ./...` 不會掃到它。

<details>
<summary>參考解答</summary>

1. **基底是 `static-debian13`，沒有 `:nonroot`**：容器以 root（UID 0）執行，我們用 `docker inspect` 確認 `Config.User` 是 `0`。出現容器逃逸漏洞時，攻擊者直接是 host 上的 root。正確寫法是 `gcr.io/distroless/static-debian13:nonroot`。
2. **`HEALTHCHECK CMD curl ...`**：這是 shell form，需要 `/bin/sh` 與 `curl`，distroless 兩個都沒有。服務本身正常，但容器會被標成 `unhealthy`，health log 是 `exec: "/bin/sh": stat /bin/sh: no such file or directory`。任何依賴健康狀態重啟或隔離容器的編排系統都會因此誤判。正確寫法是讓執行檔自己檢查：`HEALTHCHECK CMD ["/shortener", "-healthcheck"]`，或在 Kubernetes 用 `httpGet` probe。

另外，`COPY . .` 會把整個 build context 送進 Builder 階段，雖然不會進入最終映像，但搭配 `.dockerignore` 能加快建置、避免不必要的快取失效。

</details>

## 快閃卡

<details>
<summary>Distroless 映像和一般的 <code>debian</code> 映像，最主要的差別是什麼？</summary>

Distroless 只含應用程式與它的執行期相依，沒有套件管理器、shell 和一般命令列工具。一般發行版映像則是什麼都有，為「方便使用」設計。

</details>

<details>
<summary>為什麼映像裡有 <code>curl</code>、<code>apt</code>，會讓入侵的後果更嚴重？</summary>

攻擊者拿到指令執行能力之後，需要工具才能繼續：用 `curl` 下載後續程式，用 `apt` 安裝需要的套件，用 shell 互動式探索。映像裡有這些，他就不必自帶。

</details>

<details>
<summary>映像裡的套件越多，掃描器的結果會怎樣？為什麼這是問題？</summary>

報出的 CVE 越多。其中多數你的服務沒用到，團隊看久了就開始忽略警告，真正該處理的漏洞也被淹沒。

</details>

<details>
<summary>Multi-stage build 的第二個階段怎麼拿到第一個階段編出來的執行檔？</summary>

用 <code>COPY --from=build</code> 明確指定要複製的檔案。第一個階段的其他東西，包含編譯器與原始碼，都不會進入最終映像。

</details>

<details>
<summary>Dockerfile 寫 <code>ENTRYPOINT /shortener</code>，基底是 distroless。容器會怎樣？為什麼？</summary>

容器啟動失敗。Shell form 會被包成 <code>/bin/sh -c</code>，而 distroless 沒有 <code>/bin/sh</code>。要改成 exec form：<code>ENTRYPOINT ["/shortener"]</code>。

</details>

<details>
<summary>Go 服務用 <code>golang</code> 映像建置後複製進 <code>static</code> 映像，啟動時報 <code>no such file or directory</code>，檔案明明存在。原因是什麼？</summary>

執行檔是動態連結 libc 的（<code>CGO_ENABLED</code> 預設為 1），而 <code>static</code> 映像沒有 libc。建置時設 <code>CGO_ENABLED=0</code>，或改用有 glibc 的 <code>base</code>。

</details>

<details>
<summary>為什麼在 distroless 容器上 <code>kubectl exec -it pod -- sh</code> 會失敗？有什麼替代做法？</summary>

映像裡沒有 <code>sh</code>。替代做法是換成 <code>:debug</code> tag 的映像（內含 busybox），或用 <code>kubectl debug</code> 掛一個臨時的 debug 容器到同一個 Pod，後者不必動原本的映像。

</details>

<details>
<summary>容器內以 root 執行，和以 UID 65532 執行，差在哪裡？</summary>

容器內的 root 和 host 上的 root 是同一個 UID，只被 namespace 與 capabilities 圈起來。一旦出現容器逃逸漏洞，攻擊者直接就是 host root；非 root 能擋下不少這類攻擊。

</details>

<details>
<summary>什麼情況下 distroless 不是好選擇？</summary>

服務還在頻繁排錯、團隊不熟除錯手法時；或服務必須在執行期呼叫 <code>ffmpeg</code>、<code>git</code>、shell 腳本之類外部程式時。

</details>

## 延伸閱讀

- [GoogleContainerTools/distroless](https://github.com/GoogleContainerTools/distroless)：映像家族、tag 與除錯方式的最新說明，版本號以這裡為準。
- [bazel-contrib/rules_oci](https://github.com/bazel-contrib/rules_oci)：用 Bazel 以 distroless 為基底建置映像的規則。
- [Multi-stage builds（Docker 文件）](https://docs.docker.com/build/building/multi-stage/)：multi-stage Dockerfile 的語法與使用方式。
- [Ephemeral Containers（Kubernetes 文件）](https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/)：沒有 shell 的容器怎麼用臨時容器除錯。
- [CVE-2019-5736（NVD）](https://nvd.nist.gov/vuln/detail/CVE-2019-5736)：runc 容器逃逸漏洞，說明容器內 root 的風險。
