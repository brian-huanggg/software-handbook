# Linux / Bash 指令 150 題訓練營（規劃稿）

目標：從零開始熟悉 Bash 與 Linux 管理員的核心指令，再延伸到 docker、kubectl。共 150 題，分 5 個級別，每題都能自動驗證對錯。

狀態：規劃階段。本檔只有設計，尚未建立環境與題目。

## 設計原則

1. **在 Linux 容器裡練，不在 macOS 練。** macOS 是 BSD 工具，`sed -i`、`date`、`useradd` 都與 Linux 不同，而且 user 管理、`/etc`、cron 都不能動。使用 Ubuntu 24.04 容器，弄壞就重建。
2. **每題可自動驗證。** 輸出題要求把結果存到檔案，狀態題檢查系統狀態，驗證腳本與標準資料集比對。使用者不用自己對答案。
3. **同一份資料集貫穿全程。** 學員只需要熟悉一套假資料（log、CSV、專案目錄），把力氣放在指令上。
4. **指令反覆出現。** `find` 在 L1 學、L2 接 `xargs`、L4 寫進腳本、L5 用在容器內，不是學完就丟。
5. **優先順序依需求排。** 使用者最在意的 grep、curl、user、read 都排進前段並配高題數。

## 級別與章節分配

| 級別 | 題號 | 題數 | 主題 |
|---|---|---|---|
| L1 生存 | Q1–30 | 30 | 導覽、檔案操作、重導向與 pipe、quoting 與 exit code |
| L2 文字處理 | Q31–60 | 30 | grep、cut/sort/uniq/tr、sed、awk、xargs/tee/read |
| L3 Linux 管理員 | Q61–90 | 30 | user/group、權限、process、systemd、disk、cron、apt、tar |
| L4 curl、網路與腳本 | Q91–120 | 30 | curl、jq、網路診斷、Bash 腳本 |
| L5 容器與叢集 | Q121–150 | 30 | docker 10 題、kubectl 20 題 |

### L1 生存（Q1–30）

| 章節 | 題號 | 內容 |
|---|---|---|
| A 導覽與檢視 | Q1–9 | `pwd ls cd head tail wc cat`、隱藏檔、相對與絕對路徑 |
| B 檔案操作 | Q10–20 | `mkdir -p touch cp mv rm`、`find`（name/size/empty）、glob |
| C 重導向與 pipe | Q21–27 | `> >> 2> 2>&1`、pipe、`sort | uniq`、`wc -l` |
| D 變數與說明 | Q28–30 | 單雙引號、`$?`、`/dev/null`、`--help` |

產出：能在終端機裡自在移動、操作檔案、串接指令。

### L2 文字處理（Q31–60）

| 章節 | 題號 | 內容 |
|---|---|---|
| A grep | Q31–40 | `-i -v -c -n -o -r -E -w -A/-B/-C`、regex 基礎、搜 log 錯誤 |
| B 欄位與排序 | Q41–46 | `cut sort(-n -k -r) uniq -c tr wc` |
| C sed | Q47–51 | 取代 `s///`、刪行、行範圍、`-i`、`-n ... p` |
| D awk | Q52–57 | 欄位 `$1`、`-F`、條件、累加、`NR`、輸出格式 |
| E xargs / tee / read | Q58–60 | `find | xargs`、`tee`、`while read line` |

產出：能從 log 與 CSV 取出需要的資訊，並解釋 BRE 與 ERE 的差異。

### L3 Linux 管理員（Q61–90）

| 章節 | 題號 | 內容 |
|---|---|---|
| A user / group | Q61–66 | `useradd usermod userdel passwd id groups`、`/etc/passwd` 欄位、`sudo` |
| B 權限 | Q67–70 | `chmod`（數字與符號）、`chown`、`umask`、特殊權限位 |
| C process | Q71–74 | `ps top kill`、signal、背景工作 `& jobs fg` |
| D systemd 與 log | Q75–78 | `systemctl`、`journalctl`、`/var/log` |
| E disk | Q79–80 | `df du`、找出佔空間最多的目錄 |
| F cron 與環境 | Q81–84 | `crontab`、`PATH`、`alias`、`~/.bashrc` 與環境變數 |
| G 套件與封存 | Q85–88 | `apt`、`tar`、`gzip`、`rsync` |
| H 綜合 | Q89–90 | 例如「建立使用者、限定目錄權限、排程備份」 |

產出：能獨立完成一台主機的帳號、權限、服務與備份設定。

### L4 curl、網路與腳本（Q91–120）

| 章節 | 題號 | 內容 |
|---|---|---|
| A curl 基礎 | Q91–96 | `-s -S -i -I -o -O -L -w`、狀態碼、header |
| B curl 進階 | Q97–104 | `-X -H -d`、JSON、Bearer 與 Basic 認證、cookie、`--retry`、`--max-time`、`-f` |
| C jq | Q105–106 | 取欄位、過濾陣列，接在 curl 之後 |
| D 網路診斷 | Q107–111 | `ss ip ping dig nc`、`ssh scp` |
| E Bash 腳本 | Q112–118 | `if for while case`、函式、`set -euo pipefail`、`trap`、`getopts` |
| F 綜合 | Q119–120 | 寫健康檢查腳本（重試、退出碼）、分析 API 回應 |

curl 題全部打本地 mock API（`http://api:8080`），不打外網，結果可重現。mock API 提供的路由：

| 路由 | 用途 |
|---|---|
| `/health`、`/users`、`/users/<id>` | 基本 GET、404 |
| `POST /users` | JSON body、400/415/201 與 `Location` |
| `/auth/bearer`、`/auth/basic` | 認證 |
| `/redirect/<n>` | 重新導向鏈 |
| `/flaky`、`/rate-limit` | 前兩次回 503 / 429，測 `--retry` |
| `/slow?delay=n`、`/status/<code>` | timeout、狀態碼 |
| `/download/report.csv` | 下載、`-O`、`-J` |
| `/cookie/set`、`/cookie/check` | cookie jar |
| `/echo` | 回傳收到的 header 與 body，確認自己送了什麼 |

產出：能用 curl 完整操作 REST API，並把重複動作寫成可靠的腳本。

### L5 容器與叢集（Q121–150）

| 章節 | 題號 | 內容 |
|---|---|---|
| A docker | Q121–130 | `run exec logs ps stop rm`、`-p -v -e`、`build`、network、`inspect`、清理 |
| B kubectl 基礎 | Q131–138 | `get describe logs exec`、namespace、`-o yaml/json/jsonpath` |
| C kubectl 操作 | Q139–145 | `apply delete rollout scale port-forward`、ConfigMap/Secret |
| D 除錯 | Q146–150 | CrashLoopBackOff、ImagePullBackOff、Pending 的排查 |

產出：把前四級的 shell 能力帶進容器與 Kubernetes 日常維運。

## 學習路線

```
L1 生存 ──► L2 文字處理 ──► L3 管理員 ──► L4 curl 與腳本 ──► L5 容器與叢集
 檔案          grep/awk       user/權限       curl/jq           docker
 pipe          sed/xargs      process/cron    腳本化             kubectl
```

- **L1 → L2**：L1 的 pipe 與 `find` 是 L2 的基礎。先會搬資料，再學加工資料。
- **L2 → L3**：管理員工作大量讀 log 與設定檔，grep 與 awk 是進入 L3 前的必要能力。
- **L3 → L4**：L3 的權限、cron、環境變數是寫腳本的前提。腳本放在 curl 之後，因為健康檢查、重試是最自然的第一個腳本題材。
- **L4 → L5**：容器與 kubectl 的指令大多是「`get` 再 pipe 給 grep/jq/awk」。L5 假設 L2 與 L4 已經熟練。

### 反覆出現的主線

| 指令 | L1 | L2 | L3 | L4 | L5 |
|---|---|---|---|---|---|
| `find` | 找檔案 | 接 `xargs` | 找大檔、權限異常 | 寫進腳本 | 容器內找檔 |
| `grep` | `--help` 查詢 | 專章 | 查 log、設定 | 過濾 curl 輸出 | `kubectl ... | grep` |
| 重導向 | 基礎 | `tee` | cron 的 log | 腳本錯誤處理 | `logs > file` |
| exit code | `$?` | `grep` 的 0/1/2 | 服務狀態 | `set -e`、`curl -f` | 部署成功判斷 |

### 每級收尾

L2、L3、L4 最後各有 2 題綜合情境，把該級指令串起來。L5 的除錯章節就是綜合題。

## 題型

| 題型 | 驗證方式 | 範例 |
|---|---|---|
| 輸出題 | 比對 `out/Qnn.txt` 與標準答案 | 「找出出現最多次的 IP，存到檔案」 |
| 狀態題 | 檢查系統狀態 | 「建立使用者 `deploy`，加入群組 `www`」 |
| 除錯題 | 修好腳本後重新驗證 | 「這支腳本在空目錄會誤刪，找出原因」 |

L1–L2 以輸出題為主，L3 以狀態題為主，L4 起加入除錯題。

## 每題格式

1. **情境**與**需求**：說清楚輸入檔案與輸出位置。
2. **預期結果**：例如筆數、內容或狀態。
3. **提示**（可選）：卡關時才看，只給方向，不給答案。
4. **解答**（獨立放在 `answers/`）：參考指令、思路、常見錯誤，並說明為什麼不用另一種寫法（例如 `grep` 與 `awk`）。

## 預計目錄結構

```
playground/code/commands/
  README.md
  Dockerfile              # Ubuntu 24.04 與工具
  docker-compose.yml      # lab（練習）+ api（mock API）
  run.sh                  # up / shell / check Qn / checkall / reset / down
  fixtures/               # 標準資料集（log、CSV、專案目錄）
  api/server.py           # mock API，只用 Python 標準函式庫
  checks/                 # 每題一支驗證腳本與共用函式
  questions/              # L1-basics.md ... L5-containers.md
  answers/                # 對應解答
```

## 實作計劃

每個階段都要通過同一個驗收：**空白狀態全部失敗，跑完參考解答全部通過。** 這樣能確保驗證腳本既不會誤判也不會漏判。

| 階段 | 內容 | 驗收 |
|---|---|---|
| P0 環境 | Dockerfile、compose、`run.sh`、`lab-reset`、fixtures、mock API | `./run.sh up` 後能進 shell，mock API 的每條路由都能回應 |
| P1 L1 | 30 題、驗證腳本、解答 | 空白 0/30，參考解答 30/30 |
| P2 L2 | 30 題。fixtures 補上較大的 access log 與 CSV | 同上 |
| P3 L3 | 30 題。需先解決 systemd 與 root 權限（見下） | 同上 |
| P4 L4 | 30 題。用 mock API 驗證 curl 的行為 | 同上，並檢查 API 的呼叫紀錄 |
| P5 L5 | 30 題。docker 與 kubectl 環境（見下） | 同上 |

建議每個階段完成後各自提交，一個級別一個 PR，方便審查。

P0 與 P1 已在先前的實作嘗試中驗證過可行（空白 0/30，參考解答 30/30），確認後可以直接重建。

## 已知風險與待決定

| 項目 | 問題 | 建議 |
|---|---|---|
| systemd（L3-D） | 一般容器沒有 systemd，`systemctl` 不能用 | lab 容器改用以 systemd 為 PID 1 的映像，需要 `privileged` 或 `cgroupns`。若不想放寬權限，這 4 題改成讀 `journalctl` 的預錄輸出檔 |
| docker（L5-A） | 容器內要操作 docker | 掛載 host 的 `docker.sock`，或另起 DinD。掛 socket 簡單，但等於給容器 host 的 docker 權限，僅限本機練習 |
| kubectl（L5-B~D） | 需要叢集 | 用 `kind`（在 docker 內建叢集）。學員需先安裝，加入先決條件檢查 |
| 資料集共用 | 學員做壞狀態會影響後面的題目 | 每題只依賴 `fixtures/`，不依賴前一題的結果，並提供 `./run.sh reset` |
| `sudo` | 學員帳號預設免密碼 sudo，會讓 L3 的 sudoers 題失去意義 | L3 開頭改成要求密碼的設定，或讓該題自己修改 sudoers |

## 學習方法

1. 先讀完該級的「必備概念」，再開始做題。
2. 先自己寫。卡超過 10 分鐘才看提示，看完提示仍不會才看解答。
3. 看完解答後，把同一題用另一種指令再寫一次（例如 `grep` 換成 `awk`）。
4. 每做完一個級別，回頭用 `./run.sh checkall` 確認沒有退步。
5. 預估時間：平均每題 10–15 分鐘，全部約 25–35 小時。
