---
title: Golden Image
description: 建立一份受信任的基底映像清單，用 digest、掃描、簽章把關，並用 Conftest 擋下任何不從它開始的 Dockerfile。
sidebar:
  order: 4
---

## 先想想

公司有二十個團隊，每個團隊的 Dockerfile 開頭都不一樣：有人寫 `FROM golang:latest`，有人寫 `FROM alpine:3.20`，還有人從 Docker Hub 上一個不認識的帳號挑了「看起來比較小」的映像。

某天資安團隊公布一個 base image 裡的嚴重漏洞，老闆問你：「全公司有哪些服務受影響？多久能修完？」

你手上只有二十份各自為政的 Dockerfile。你要怎麼回答這兩個問題？如果從一開始就規定「所有服務只能從同一份核准的基底開始」，這兩個問題會變成什麼樣子？

<details>
<summary>提示</summary>

想想「找出誰受影響」和「修好一次」各自需要什麼：一份清單、一個統一的修補點，還有一個能確保大家真的照做的機制。

</details>

## 是什麼

**Golden Image** 是由平台團隊建置、掃描、簽章並持續維護的一組核准 base image，組織內的服務一律從它開始建置，其他來源的 `FROM` 在 CI 就會被擋下。

它不是某個特定映像，而是一套做法：一份「誰能當基底」的白名單，加上讓白名單有意義的四個動作。和上一篇的 [distroless 映像](/devops/l1/1-2-container-image/distroless-image/) 的關係是：distroless 回答「基底裡該放什麼」，Golden Image 回答「誰來決定基底、怎麼確保大家都用它」。Golden Image 通常就是建立在 distroless 或其他最小基底之上，再加上組織自己需要的東西，例如內部 CA 憑證。

## 為什麼存在

沒有 Golden Image 的時候，每個團隊自己挑基底，有三個後果。

**修補是 N 次，不是 1 次。** 基底出現漏洞時，每個團隊要各自發現、各自評估、各自升級。通報到修完的時間，取決於最慢的那個團隊。

**盤點做不到。** 「全公司誰在用 OpenSSL 3.0.x」這個問題，要查二十個 repo 才有答案，而且你還不確定有沒有漏掉。

**信任沒有來源。** 公開 registry 上的映像，維護者可能換人、帳號可能被盜、tag 可能被改指。你的服務啟動在一個沒人審過的東西上。

Golden Image 把這件事收斂到一個地方：平台團隊負責基底的內容、漏洞與更新節奏，應用團隊只負責自己的程式。這和不讓每個團隊自己寫 TLS 實作是同一種分工。

## 怎麼運作

:::note
本篇的範例都在 `playground/devops/l1/1-2-container-image/golden-image/`。Conftest 範例需要安裝 [conftest](https://www.conftest.dev/)，建置 golden image 需要 Docker。掃描與簽章的指令只示範用法，本篇沒有實際執行。
:::

Golden Image 靠四道關卡串成一條線，缺任何一道，後面的就失去意義：

```text
 核准              固定              驗證                強制
┌──────────┐    ┌──────────┐    ┌──────────────┐    ┌──────────────┐
│ 平台團隊  │    │ digest   │    │ 掃描 + 簽章   │    │ CI 擋 FROM    │
│ 決定基底  │ ─► │ @sha256  │ ─► │ 發布時做一次  │ ─► │ 叢集擋未簽章  │
│ 內容與清單│    │ 內容不可變│    │ 使用時驗證    │    │ 沒有後門      │
└──────────┘    └──────────┘    └──────────────┘    └──────────────┘
 誰說了算         指的是哪一份      這份可不可信         走不走得過去
```

由左到右看，每一格回答一個問題：誰決定、指的是哪一份、這份可不能信、不照做會怎樣。

### 核准：誰來決定基底

平台團隊建置 golden image，Dockerfile 本身就是政策的一部分：

```dockerfile title="golden/Dockerfile"
# The golden image is the one place a public image is allowed in FROM:
# it is the origin, so it is pinned by digest and rebuilt on a schedule.
FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329
LABEL org.opencontainers.image.title="go-static" \
      org.opencontainers.image.vendor="internal-platform-team"
```

這是整個組織唯一允許直接引用公開映像的地方。一份實用的 golden image 清單通常很短：一個 Go 建置用的 `go-builder`、一個執行用的 `go-static`、也許再加一個 Python 與 Node.js。種類越少，維護越輕。

除了內容，「核准」還包含節奏：多久重建一次（例如每週，或基底有嚴重漏洞時立即重建）、舊版保留多久、什麼時候停用。這些要寫下來，否則 Golden Image 會變成沒人更新的另一個過期映像。

### 固定：用 digest 指向確切的內容

**tag** 是可以被改指的名稱，**digest** 是內容的雜湊值。同一個 tag 今天和明天可能指向不同的映像，同一個 digest 永遠是同一份內容。取得 digest 的方法：

```bash
docker buildx imagetools inspect gcr.io/distroless/static-debian13:nonroot
```

```text
Name:      gcr.io/distroless/static-debian13:nonroot
MediaType: application/vnd.oci.image.index.v1+json
Digest:    sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329
```

注意 `MediaType` 是 image index：這個 digest 指向的是「多架構的目錄」，拉取時 Docker 再依你的平台挑對應的那一份。固定 index 的 digest，在 amd64 與 arm64 上拿到的是同一次發布的內容。

應用團隊的 Dockerfile 引用 golden image 時也寫 digest：

```dockerfile title="good.Dockerfile"
FROM registry.internal.example/golden/go-builder@sha256:1111…1111 AS build
# ...
FROM registry.internal.example/golden/go-static@sha256:2222…2222
COPY --from=build /out/shortener /shortener
ENTRYPOINT ["/shortener"]
```

（範例檔案裡是完整的 64 位十六進位字元，這裡為了排版省略。）tag 與 digest 的差別，以及 tag 被改指的實驗，在資安領域的 [Container 與 Image 的攻擊面](/cyber/p4-advanced/4-3-cloud-container/container-image-attack-surface/) 已經用本機重現過，這裡不重複。

固定 digest 的代價是：你不會再「自動」拿到修補。這個取捨在後面的「取捨與替代方案」說明。

### 驗證：掃描與簽章

Golden image 在發布前要過兩件事。

**掃描**：用漏洞掃描器比對映像內的套件。重點是把它當成發布門檻，而不是事後報表：

```bash
trivy image --severity HIGH,CRITICAL --ignore-unfixed --exit-code 1 \
  registry.internal.example/golden/go-static@sha256:2222…2222
```

映像有已經能修補的 HIGH 或 CRITICAL 漏洞時，`--exit-code 1` 讓指令失敗，CI 就不會把這份映像標為核准。`--ignore-unfixed` 排除「上游還沒有修補版」的漏洞，因為那種你什麼也做不了，列進門檻只會讓發布永遠卡住。

**簽章**：掃描通過之後，用 [cosign](https://github.com/sigstore/cosign) 對這個 digest 簽章，等於平台團隊蓋章說「這一份是我們核准的」。也可以一併附上 SBOM（軟體物料清單，列出映像內所有元件）。簽章與驗證的指令如下：

```bash
# Platform team, at publish time.
cosign sign --yes registry.internal.example/golden/go-static@sha256:2222…2222

# Anyone, at use time.
cosign verify \
  --certificate-identity-regexp 'https://github.com/example/platform/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  registry.internal.example/golden/go-static@sha256:2222…2222
```

簽章與驗證的完整串接，包含 keyless 簽章與在 CI 裡怎麼放，留到 1.3 的流水線再做。這裡先記住兩件事：簽的對象是 digest，而不是 tag；簽章沒有人驗證，等於沒簽。

### 強制：只能從它開始

前三步都只是「約定」。沒有第四步，總會有人為了趕時間直接寫 `FROM python:3`。強制可以放在三個位置，越往後越晚攔截：

| 位置 | 做法 | 何時擋下 |
| --- | --- | --- |
| 開發與 CI | 用 Conftest 檢查 Dockerfile 的每個 `FROM` | PR 階段，最早也最便宜 |
| Registry | 只允許從內部 registry 拉取，對外封閉 | 建置階段 |
| 叢集 | Admission control 驗證映像簽章，拒絕未簽章或不在白名單的映像 | 部署階段，最後一道 |

本篇只實作第一層。叢集那層（例如 Kyverno 的 `verifyImages`，或 Sigstore 的 policy-controller）要等到 L3 叢集營運學了 Kubernetes 才適合展開；這裡你只需要知道它的位置：就算有人繞過 CI，沒有簽章的映像也進不了叢集。

[Conftest](https://www.conftest.dev/) 用 Rego 語言寫政策，能把 Dockerfile 解析成指令清單再逐條檢查。核心規則是：`FROM` 必須以 golden 前綴開頭，而且要有 `@sha256:`：

```rego title="policy/dockerfile.rego"
deny contains msg if {
	some instruction in input
	instruction.Cmd == "from"
	ref := instruction.Value[0]
	not lower(ref) in stage_names
	not startswith(ref, golden_prefix)
	msg := sprintf("FROM %s is not a golden image (must start with %s)", [ref, golden_prefix])
}

deny contains msg if {
	some instruction in input
	instruction.Cmd == "from"
	ref := instruction.Value[0]
	startswith(ref, golden_prefix)
	not contains(ref, "@sha256:")
	msg := sprintf("FROM %s is not pinned by digest (add @sha256:...)", [ref])
}
```

`stage_names` 是同一個 Dockerfile 裡用 `AS` 命名的階段，讓 `FROM build` 這種「接著上一個階段」的寫法不被誤判。拿一份用公開映像的 Dockerfile 試試：

```bash
conftest test --parser dockerfile -p policy testdata/public-latest.Dockerfile
```

```text
FAIL - testdata/public-latest.Dockerfile - main - FROM debian:13-slim is not a golden image (must start with registry.internal.example/golden/)
FAIL - testdata/public-latest.Dockerfile - main - FROM golang:latest is not a golden image (must start with registry.internal.example/golden/)
4 tests, 2 passed, 0 warnings, 2 failures, 0 exceptions
```

Conftest 在有 `deny` 時以非 0 結束，CI 就會失敗，PR 不能合併。

回到一開始的問題：二十份各自為政的 Dockerfile，你無法回答「誰受影響」；如果規定了只能從 golden image 開始，答案是查「哪些映像的 digest 來自受影響的那一版」，因為每個合法的 `FROM` 都會寫出確切的 digest。修補則變成：平台團隊重建一次 golden image，應用團隊把 `FROM` 的 digest 換成新的（可以由機器人自動開 PR），重新建置即可。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 直接用公開映像的 tag（`golang:1.26`） | 零維護，永遠拿到最新的修補 | 內容會悄悄改變；建置不可重現；沒有統一管控 | 個人專案、原型 |
| 公開映像加 digest | 建置可重現，不會被改指 | 沒有人替你重建，修補要自己追；每個團隊各做各的 | 團隊只有一兩個服務 |
| Golden Image：內部 registry、digest、掃描、簽章、強制 | 修補集中；能盤點；信任有來源 | 平台團隊要長期維護；它壞了所有人一起壞 | 多個團隊共用基底的組織 |
| 每個服務從零建基底 | 完全掌控內容 | 重複勞動；每個團隊都要有資安能力 | 有特殊合規需求的單一服務 |

**固定 digest 與自動更新的矛盾。** Tag 會自己跟著上游走，這是它的缺點也是它的優點；digest 固定住了，修補就不會自動進來。解法是把「更新」變成一個可審查的動作：用 [Renovate](https://docs.renovatebot.com/docker/) 或 Dependabot 這類工具，在 golden image 發布新版時自動對每個 repo 開 PR，把 digest 換成新的。你同時得到可重現的建置與及時的修補，代價是要處理這些 PR。

**單點風險。** Golden Image 讓信任集中，也讓失誤集中：基底建壞了，所有人一起壞。對策是發布前先在少數服務上試跑，並且保留前幾版 digest 讓人能回退。

**什麼時候不該用。**

- 只有一兩個服務、一個團隊：維護 golden image 和簽章流程的成本超過收益。公開映像加 digest 加定期掃描就夠。
- 服務有組織通用基底無法滿足的需求（例如需要特定的 GPU 驅動）：硬塞進通用 golden image 會讓它變肥。這時應該有明確的例外流程，由平台團隊另外核准一個專用映像，而不是讓團隊自行繞開。

## 常見坑與故障案例

### 固定了 digest，然後再也沒更新

**症狀**：半年後掃描器報出一長串漏洞，而且都出在基底。**原因**：digest 讓建置不會變，也讓基底不會變。沒有人更新 digest，映像就停在半年前的狀態。（示意案例）**對策**：把更新自動化。Renovate 開 PR，golden image 設定重建排程，並在掃描結果中標示「基底的 digest 已落後幾版」。

### 只檢查 `FROM`，被別的寫法繞過

**症狀**：CI 的 `FROM` 檢查全部通過，映像裡卻出現了不在白名單的東西。**原因**：Dockerfile 有不止一個地方可以拉取映像。`COPY --from=docker.io/library/busybox:latest ...` 會直接從公開映像複製檔案；`ARG BASE=...` 搭配 `FROM ${BASE}` 讓實際的映像在建置時才決定，政策看不到值。**對策**：政策裡明確禁止 `FROM` 使用變數，並檢查 `COPY --from` 的來源必須是前面的階段名稱。我們的 Conftest 政策已經包含這兩條：

```rego title="policy/dockerfile.rego"
deny contains msg if {
	some instruction in input
	instruction.Cmd == "copy"
	some flag in instruction.Flags
	startswith(flag, "--from=")
	source := lower(trim_prefix(flag, "--from="))
	not source in stage_names
	not regex.match(`^[0-9]+$`, source)
	msg := sprintf("COPY %s pulls an image outside FROM; use a golden image stage instead", [flag])
}
```

### 簽了章，但沒有人驗證

**症狀**：所有 golden image 都有簽章，但有人推了一份沒簽章的映像上線，沒人發現。**原因**：簽章只是證明「誰核准了這份內容」，不會自己擋任何東西。真正的保護來自驗證的那一端。**對策**：在部署階段由 admission control 驗證簽章，把「沒有簽章」變成部署失敗。簽的時候也要簽 digest，簽 tag 的話，tag 被改指後簽章就失去對象。

### 上游出事，全公司一起中

**症狀**：公開的基底映像被發現有嚴重問題，而你的 golden image 就是從它來的。**原因**：Golden Image 把信任集中，也把風險集中。真實案例是 2019 年的 [CVE-2019-5021](https://nvd.nist.gov/vuln/detail/CVE-2019-5021)：多個版本的 Alpine Linux 官方 Docker 映像，root 帳號的密碼是空的。所有從這些版本建出來的映像都帶著這個問題，而且多數使用者並不知道。**對策**：Golden Image 的好處在這時候才顯現：因為每個服務的 `FROM` 都寫出確切的 digest，你可以查出誰用了受影響的那一份；平台團隊重建一次，其他團隊換 digest 即可。前提是這份清單真的存在，而且有人在看上游的公告。

## 小練習

### 動手寫

目前的政策只檢查「是不是 golden image、有沒有 digest」。假設某一版 golden image 被發現有嚴重漏洞，平台團隊把它的 digest 列為撤銷。請補完 `exercise/policy/revoked.rego`：寫一條 `deny` 規則，拒絕任何 `FROM` 的 digest 落在 `revoked_digests` 裡的 Dockerfile。

起始程式碼在 `playground/devops/l1/1-2-container-image/golden-image/exercise/`。執行下面的指令，做對時兩個測試都會通過：

```bash
conftest verify -p exercise/policy
```

<details>
<summary>參考解答</summary>

```rego title="revoked.rego"
deny contains msg if {
	some instruction in input
	instruction.Cmd == "from"
	digest := split(instruction.Value[0], "@")[1]
	digest in revoked_digests
	msg := sprintf("FROM %s uses a revoked digest", [instruction.Value[0]])
}
```

`split(..., "@")[1]` 取出 `@` 後面的 digest。沒有 `@` 的 `FROM`（例如接著上一個階段的 `FROM build`）會讓 `[1]` 沒有值，這條規則就不成立，不會誤擋。這類 `FROM` 本來就由前面的規則負責。

</details>

### 辨識

下面這份 Dockerfile 由 AI 產生，目的是「讓短網址服務只從組織的 golden image 建置」。它能建置，映像也能正常啟動。請做 code review，找出它違反 golden image 原則的地方（至少兩個）。

```dockerfile title="Dockerfile"
ARG GO_VERSION=1.26
FROM registry.internal.example/golden/go-builder:${GO_VERSION} AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/shortener .

FROM registry.internal.example/golden/go-static:nonroot
COPY --from=build /out/shortener /shortener
COPY --from=docker.io/library/busybox:latest /bin/busybox /bin/busybox
ENTRYPOINT ["/shortener"]
```

完整檔案在 `recognize/Dockerfile`，`go test ./...` 不會掃到它。

<details>
<summary>參考解答</summary>

1. **兩個 `FROM` 都只用 tag，沒有 digest**：`go-builder:${GO_VERSION}` 與 `go-static:nonroot` 看起來是 golden image，但 tag 可以被改指，建置不可重現，被入侵的 registry 帳號也能悄悄換掉內容。正確寫法是 `registry.internal.example/golden/go-static@sha256:…`。
2. **`FROM` 的 tag 由 `ARG` 決定**：實際拉取的映像要到建置時才確定，政策檢查與盤點都看不到最終的值。正確寫法是把版本直接寫在 `FROM` 裡（並且用 digest）。
3. **`COPY --from=docker.io/library/busybox:latest`**：這一行繞過了 `FROM` 的白名單，直接從公開映像拿檔案，而且用的是 `latest`。更糟的是，它把 busybox 放進最小基底，等於把上一篇辛苦拿掉的 shell 又裝回去。正確做法是不要這一行；需要除錯用的 shell，用 `kubectl debug` 掛臨時容器。

跑 `conftest test --parser dockerfile -p policy recognize/Dockerfile`，政策會同時報出這幾項。

</details>

## 快閃卡

<details>
<summary>Golden Image 是什麼？它和 distroless 映像是什麼關係？</summary>

Golden Image 是由平台團隊建置、掃描、簽章並維護的一組核准 base image，組織內的服務一律從它開始。Distroless 決定基底裡放什麼，Golden Image 決定誰能當基底；Golden Image 常常就建立在 distroless 之上。

</details>

<details>
<summary>沒有 Golden Image 時，基底出現漏洞，為什麼修補慢、也很難盤點？</summary>

每個團隊各自挑基底，所以要各自發現、各自升級，完成時間取決於最慢的團隊。也沒有一份統一清單，無法回答「誰在用受影響的版本」。

</details>

<details>
<summary>tag 和 digest 差在哪？為什麼 Golden Image 要用 digest？</summary>

Tag 是可以被改指的名稱，同一個 tag 今天與明天可能是不同的映像。Digest 是內容的雜湊值，同一個 digest 永遠是同一份內容，所以建置可重現，也不會被悄悄換掉。

</details>

<details>
<summary>Dockerfile 寫 <code>FROM registry.internal.example/golden/go-static:1.4</code>，用本文的 Conftest 政策檢查，會通過嗎？</summary>

不會。它以 golden 前綴開頭，通過了「是不是 golden image」，但沒有 <code>@sha256:</code>，會被「必須用 digest 固定」那條規則擋下。

</details>

<details>
<summary>固定了 digest 之後，基底的漏洞修補還會自動進來嗎？怎麼補救？</summary>

不會，digest 固定了就不會變。補救是把更新自動化：golden image 發布新版時，由 Renovate 或 Dependabot 對每個 repo 開 PR，把 digest 換成新的。

</details>

<details>
<summary>Dockerfile 裡有 <code>ARG BASE=...</code> 與 <code>FROM ${BASE}</code>，為什麼會讓政策失效？</summary>

實際的映像要到建置時才由變數決定，靜態檢查看到的只是 <code>${BASE}</code>，無法判斷它是不是 golden image。所以政策應該直接禁止 <code>FROM</code> 使用變數。

</details>

<details>
<summary>CI 的 <code>FROM</code> 檢查全部通過，映像裡還是可能出現未核准的內容。舉一個 Dockerfile 的例子。</summary>

<code>COPY --from=docker.io/library/busybox:latest ...</code> 會直接從公開映像複製檔案，完全沒經過 <code>FROM</code>。所以政策還要檢查 <code>COPY --from</code> 的來源只能是前面的階段。

</details>

<details>
<summary>所有 golden image 都簽了章，為什麼還是可能有未核准的映像上線？</summary>

簽章只證明「誰核准了這份內容」，本身不擋任何東西。要在部署階段由 admission control 驗證簽章，沒簽章就拒絕，保護才算生效。

</details>

<details>
<summary>什麼情況下不該為組織建 Golden Image？</summary>

只有一兩個服務、一個團隊時，維護成本超過收益；或服務有通用基底滿足不了的特殊需求時，應該走例外流程另外核准專用映像，而不是把通用基底做肥。

</details>

## 延伸閱讀

- [Conftest](https://www.conftest.dev/)：用 Rego 對 Dockerfile、Kubernetes manifest 等設定檔寫政策並在 CI 執行。
- [Trivy](https://github.com/aquasecurity/trivy)：映像漏洞掃描器，支援 `--exit-code` 當作發布門檻。
- [cosign](https://github.com/sigstore/cosign)：Sigstore 的映像簽章與驗證工具，1.3 的流水線會用到。
- [Renovate：Docker 支援](https://docs.renovatebot.com/docker/)：自動更新 Dockerfile 中固定 digest 的設定方式。
- [Kyverno](https://kyverno.io/)：Kubernetes 的政策引擎，可以在叢集端驗證映像簽章。
- [Docker 建置最佳實務](https://docs.docker.com/build/building/best-practices/)：官方對固定基底映像版本與 digest 的建議。
- [CVE-2019-5021（NVD）](https://nvd.nist.gov/vuln/detail/CVE-2019-5021)：Alpine 官方映像 root 密碼為空的漏洞，說明基底問題如何擴散。
