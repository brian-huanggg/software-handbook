# 計算機科學目錄

這是全書的共同基礎，寫給自學的初級工程師。後端、DevOps、資安三個領域都假設你讀過這裡的對應章節。
自學者最常缺的就是底層知識，所以這個領域要寫厚：目標是讓你能講清楚「一個請求從瀏覽器到伺服器」沿途的每一層在做什麼。

本領域**依主題分組**，不分 L0–L4；每一組都從零講起，不假設任何背景。主要語言是 Go，少數底層章節會用一點 C。
沒有貫穿專案，只有一個收尾綜合題。

| 主題 | 讀完你能做到 |
| --- | --- |
| 計算機組成與程式執行 | 說清楚程式從原始碼到在 CPU 上執行的每一步 |
| 資料結構與演算法 | 選對資料結構，估得出一段程式的時間複雜度 |
| 作業系統 | 解釋 process、記憶體、I/O 與並行背後發生的事 |
| 計算機網路 | 追蹤一個封包從你的機器到對端的完整路徑 |
| 命令列與開發工具 | 在終端機裡自在地操作、除錯與版本控制 |

寫目錄時沿用本檔格式：`##` 為主題分組、`### <節號> <名稱>` 為節、`- [ ]` 為文章。
每新增一節，同時在 `site/src/sidebar/cs.ts` 加上該節的標籤與目錄名稱。

## C 計算機組成與程式執行

### C.1 程式怎麼跑

- [ ] 編譯、直譯、位元組碼與執行檔的差別
- [ ] 從原始碼到執行檔：前處理、編譯、組合、連結
- [ ] 作業系統怎麼載入並啟動一個程式
- [ ] 以 Go 為例：`go build` 到底產出了什麼

### C.2 資料在機器裡的樣子

- [ ] 位元、位元組、位元組順序（endianness）
- [ ] 整數的表示與溢位、浮點數的陷阱
- [ ] 字元編碼：ASCII、Unicode、UTF-8
- [ ] 文字、數字、布林在記憶體裡怎麼擺

### C.3 CPU 與記憶體階層

- [ ] CPU 怎麼執行指令：取指、解碼、執行
- [ ] 暫存器、快取（L1/L2/L3）、主記憶體、磁碟的速度差距
- [ ] 快取行與區域性：為什麼存取順序會影響效能
- [ ] 為什麼後端要懂這些：資料結構的實際成本

## D 資料結構與演算法

### D.1 複雜度分析

- [ ] 時間與空間複雜度、Big-O 記號
- [ ] 最壞、平均、分攤分析
- [ ] 怎麼對一段真實程式估算複雜度

### D.2 基礎資料結構

- [ ] 陣列與動態陣列（slice）
- [ ] 鏈結串列、堆疊、佇列
- [ ] 雜湊表：雜湊函式、碰撞、負載因子
- [ ] 樹：二元搜尋樹、平衡樹、B/B+ 樹
- [ ] 堆（heap）與優先佇列
- [ ] 圖的表示法

### D.3 基礎演算法

- [ ] 排序：比較排序與其下界、常見實作
- [ ] 搜尋：線性、二元搜尋
- [ ] 遞迴與分治
- [ ] 動態規劃入門
- [ ] 圖的走訪：BFS、DFS

### D.4 把它連到後端

- [ ] 索引為什麼是 B+ 樹、快取為什麼是雜湊表
- [ ] 常見場景怎麼選資料結構

## O 作業系統

### O.1 process 與 thread

- [ ] process 與 thread 的差別、各自的成本
- [ ] 使用者態與核心態、context switch
- [ ] process 的生命週期與狀態

### O.2 記憶體管理

- [ ] 虛擬記憶體與位址空間
- [ ] stack 與 heap、page、page fault
- [ ] 記憶體配置與回收的基本概念

### O.3 排程

- [ ] CPU 怎麼在工作之間切換
- [ ] 常見排程策略與取捨
- [ ] 這如何影響延遲與吞吐

### O.4 檔案與 I/O

- [ ] 檔案系統與 inode、file descriptor
- [ ] I/O 模型：阻塞、非阻塞、多工（select/poll/epoll）
- [ ] 系統呼叫是什麼、為什麼有成本
- [ ] 資源限制：ulimit、open files

### O.5 並行基礎

- [ ] race condition 與臨界區
- [ ] lock、死結（deadlock）、飢餓
- [ ] 這些概念在 Go 並行章節會怎麼再出現

### O.6 隔離與容器的底層

- [ ] namespace 與 cgroup：容器為什麼能隔離
- [ ] 這條線接到 DevOps 與資安的容器章節

## N 計算機網路

### N.1 分層模型

- [ ] OSI 與 TCP/IP 分層、各層職責
- [ ] 一個封包怎麼被層層封裝與解封裝

### N.2 連結層與網路層

- [ ] MAC、交換、ARP 概念
- [ ] IP、子網路（subnet）、路由的基本概念
- [ ] NAT 與私有位址

### N.3 傳輸層

- [ ] port 與 socket
- [ ] TCP：三次握手、重傳、流量與壅塞控制
- [ ] UDP 與它適合的情境

### N.4 應用層

- [ ] DNS 怎麼解析一個網域
- [ ] HTTP/1.1、HTTP/2、HTTP/3 的差異
- [ ] TLS 與 HTTPS 怎麼保護連線
- [x] Origin 與同源政策：瀏覽器怎麼劃出信任邊界（CORS 主線 1/3，接後端 1.4）→ [文章](../../site/src/content/docs/cs/n-network/n-4-application/same-origin-policy.md)
- [ ] 代理、負載平衡、CDN 的角色
- [ ] WebSocket 與長連線

### N.5 動手看封包

- [ ] 用 Go 寫一個 TCP echo server
- [ ] 用 tcpdump/Wireshark 觀察一次握手與一次請求

## T 命令列與開發工具

### T.1 Linux 命令列

- [ ] 檔案系統導覽：路徑、目錄結構（`/etc`、`/var`、`/usr`、`/home`）、常用檔案操作
- [ ] pipe、重導向、grep、find、curl
- [ ] process 與訊號、看 log

### T.2 使用者、群組與權限

本節講「誰能對什麼做什麼」在 Linux 上怎麼被決定與管理。檔案系統與 file descriptor 的底層在 O.4；容器的隔離在 O.6；應用層的 RBAC 在後端 2.6 與 4.2，本節只講作業系統這一層。

- [ ] 使用者與群組模型：UID/GID、`/etc/passwd`、`/etc/shadow`、`/etc/group`、root、系統帳號與一般帳號
- [ ] 管理帳號：`useradd`、`usermod`、`userdel`、`groupadd`、`id`、`whoami`、主要群組與附加群組
- [ ] 檔案權限矩陣：owner／group／other × read／write／execute、`ls -l` 讀法、`chmod`（符號與八進位）、`chown`、`chgrp`
- [ ] 目錄權限的語意：目錄上的 `r` 與 `x` 差在哪、路徑逐層檢查、為什麼「刪檔」看的是父目錄
- [ ] 預設權限：`umask` 如何決定新檔案與新目錄的權限
- [ ] 特殊權限位：setuid、setgid、sticky bit（`/tmp`、`passwd` 為什麼能運作）與其安全風險
- [ ] 提權：`su`、`sudo`、`/etc/sudoers` 與 `visudo`、最小授權寫法、`sudo` 的稽核紀錄
- [ ] 細粒度授權：POSIX ACL（`getfacl`／`setfacl`）、什麼時候該用 ACL 而不是多開一個群組
- [ ] Linux capabilities：把 root 的權力拆開、`CAP_NET_BIND_SERVICE` 等常見能力、`getcap`／`setcap`
- [ ] 核心如何做權限檢查：process 的 real／effective／saved UID、`fork` 與 `exec` 時的身分繼承
- [ ] 服務身分：用專屬的非 root 帳號跑服務、systemd `User=`、檔案與 socket 的擁有者、為什麼不該用 root 跑網路服務
- [ ] 登入與認證：PAM 概觀、SSH 金鑰登入、`~/.ssh` 權限、`authorized_keys`、停用密碼登入
- [ ] 權限問題排查：`Permission denied` 的檢查順序（`namei -l`、`id`、`ls -ld`、`getfacl`、`strace`）
- [ ] 設計一份權限矩陣：用多人共用目錄的案例（開發者／維運／唯讀稽核）把群組、ACL、`sudo` 規則組合起來並驗證
- [ ] 容器裡的使用者：容器內 UID 與主機 UID 的對應、non-root image、掛載目錄的擁有者問題（連到 O.6 與 DevOps 容器章節）

### T.3 Shell 指令碼

- [ ] 變數、條件、迴圈、函式
- [ ] 寫一個可重複執行的小腳本

### T.4 版本控制 Git

- [ ] commit、branch、merge、rebase
- [ ] 解 conflict、閱讀歷史
- [ ] PR 流程與協作

### T.5 除錯與建置

- [ ] 用 log 與除錯器定位問題
- [ ] strace 與 tcpdump 入門
- [ ] 編輯器/IDE、Makefile

## Z 綜合題

### Z.1 把它串起來

- [ ] 「在瀏覽器輸入網址按 Enter」到底發生了什麼：從鍵盤、DNS、TCP、TLS、HTTP 到伺服器回應，走一遍所有前面學過的層
