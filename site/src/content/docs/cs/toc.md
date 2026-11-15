---
title: 計算機科學目錄
description: 計算機科學領域所有章節一覽，包含尚未寫好的章節。
sidebar:
  order: 0
---

<!-- 由 scripts/gen-toc.mjs 從 docs/contents/ 產生，請勿手動修改；改目錄請改 docs/contents/ 後重跑 `pnpm toc`。 -->

下列為計算機科學領域規劃的所有章節。左側欄只列出已寫好的章節，其餘陸續補上。

## C 計算機組成與程式執行

### C.1 程式怎麼跑

| 標題 | 說明 |
| --- | --- |
| 編譯、直譯、位元組碼與執行檔 | 四者的差別 |
| 從原始碼到執行檔 | 前處理、編譯、組合、連結 |
| 作業系統載入並啟動程式 |  |
| go build 產出了什麼 | 以 Go 為例 |

### C.2 資料在機器裡的樣子

| 標題 | 說明 |
| --- | --- |
| 位元、位元組與位元組順序 | endianness |
| 整數、溢位與浮點數 | 整數的表示與溢位、浮點數的陷阱 |
| 字元編碼 | ASCII、Unicode、UTF-8 |
| 文字、數字、布林的記憶體擺放 |  |

### C.3 CPU 與記憶體階層

| 標題 | 說明 |
| --- | --- |
| CPU 執行指令 | 取指、解碼、執行 |
| 記憶體階層的速度差距 | 暫存器、快取（L1/L2/L3）、主記憶體、磁碟 |
| 快取行與區域性 | 為什麼存取順序會影響效能 |
| 資料結構的實際成本 | 為什麼後端要懂這些 |

## D 資料結構與演算法

### D.1 複雜度分析

| 標題 | 說明 |
| --- | --- |
| 時間與空間複雜度 | Big-O 記號 |
| 最壞、平均、分攤分析 |  |
| 估算真實程式的複雜度 |  |

### D.2 基礎資料結構

| 標題 | 說明 |
| --- | --- |
| 陣列與動態陣列 | slice |
| 鏈結串列、堆疊、佇列 |  |
| 雜湊表 | 雜湊函式、碰撞、負載因子 |
| 樹 | 二元搜尋樹、平衡樹、B/B+ 樹 |
| 堆與優先佇列 | heap |
| 圖的表示法 |  |

### D.3 基礎演算法

| 標題 | 說明 |
| --- | --- |
| 排序 | 比較排序與其下界、常見實作 |
| 搜尋 | 線性、二元搜尋 |
| 遞迴與分治 |  |
| 動態規劃入門 |  |
| 圖的走訪 | BFS、DFS |

### D.4 把它連到後端

| 標題 | 說明 |
| --- | --- |
| 索引與快取的資料結構 | 索引為什麼是 B+ 樹、快取為什麼是雜湊表 |
| 常見場景怎麼選資料結構 |  |

## O 作業系統

### O.1 process 與 thread

| 標題 | 說明 |
| --- | --- |
| process 與 thread 的差別 | 各自的成本 |
| 使用者態與核心態 | context switch |
| process 的生命週期與狀態 |  |

### O.2 記憶體管理

| 標題 | 說明 |
| --- | --- |
| 虛擬記憶體與位址空間 |  |
| stack 與 heap | page、page fault |
| 記憶體配置與回收 | 基本概念 |

### O.3 排程

| 標題 | 說明 |
| --- | --- |
| CPU 怎麼在工作之間切換 |  |
| 排程策略與取捨 | 常見排程策略 |
| 排程對延遲與吞吐的影響 |  |

### O.4 檔案與 I/O

| 標題 | 說明 |
| --- | --- |
| 檔案系統與 fd | inode、file descriptor |
| I/O 模型 | 阻塞、非阻塞、多工（select/poll/epoll） |
| 系統呼叫的成本 | 系統呼叫是什麼、為什麼有成本 |
| 資源限制 | ulimit、open files |

### O.5 並行基礎

| 標題 | 說明 |
| --- | --- |
| race condition 與臨界區 |  |
| lock、死結與飢餓 | deadlock |
| 並行概念與 Go 的連結 | 這些概念在 Go 並行章節會再出現 |

### O.6 隔離與容器的底層

| 標題 | 說明 |
| --- | --- |
| namespace 與 cgroup | 容器為什麼能隔離 |
| 容器隔離的後續章節 | 接到 DevOps 與資安的容器章節 |

## N 計算機網路

### N.1 分層模型

| 標題 | 說明 |
| --- | --- |
| OSI 與 TCP/IP 分層 | 各層職責 |
| 封包的封裝與解封裝 | 一個封包怎麼被層層封裝與解封裝 |

### N.2 連結層與網路層

| 標題 | 說明 |
| --- | --- |
| MAC、交換與 ARP | 概念 |
| IP、子網路與路由 | subnet、路由的基本概念 |
| NAT 與私有位址 |  |

### N.3 傳輸層

| 標題 | 說明 |
| --- | --- |
| port 與 socket |  |
| TCP | 三次握手、重傳、流量與壅塞控制 |
| UDP | 它適合的情境 |

### N.4 應用層

| 標題 | 說明 |
| --- | --- |
| DNS 解析 | DNS 怎麼解析一個網域 |
| HTTP 各版本差異 | HTTP/1.1、HTTP/2、HTTP/3 |
| TLS 與 HTTPS | 怎麼保護連線 |
| [Origin 與同源政策](/cs/n-network/n-4-application/same-origin-policy/) | 判斷兩個網址是否同源，並說清楚瀏覽器擋的是什麼、不擋什麼。 |
| 代理、負載平衡與 CDN | 它們的角色 |
| WebSocket 與長連線 |  |

### N.5 動手看封包

| 標題 | 說明 |
| --- | --- |
| TCP echo server | 用 Go 實作 |
| 用 tcpdump 看封包 | 用 tcpdump/Wireshark 觀察一次握手與一次請求 |

## T 命令列與開發工具

### T.1 Linux 命令列

| 標題 | 說明 |
| --- | --- |
| 檔案系統導覽 | 路徑、目錄結構（`/etc`、`/var`、`/usr`、`/home`）、常用檔案操作 |
| pipe、重導向與文字工具 | grep、find、curl |
| process、訊號與看 log |  |

### T.2 使用者、群組與權限

本節講「誰能對什麼做什麼」在 Linux 上怎麼被決定與管理。檔案系統與 file descriptor 的底層在 O.4；容器的隔離在 O.6；應用層的 RBAC 在後端 2.6 與 4.2，本節只講作業系統這一層。

| 標題 | 說明 |
| --- | --- |
| 使用者與群組模型 | UID/GID、`/etc/passwd`、`shadow`、`group`、root、系統帳號 |
| 管理帳號 | `useradd`、`usermod`、`userdel`、`groupadd`、`id`、`whoami`、附加群組 |
| 檔案權限矩陣 | owner／group／other × r／w／x、`ls -l`、`chmod`、`chown`、`chgrp` |
| 目錄權限的語意 | `r` 與 `x` 的差別、路徑逐層檢查、刪檔看父目錄 |
| 預設權限與 umask | umask 如何決定新檔案與新目錄的權限 |
| 特殊權限位 | setuid、setgid、sticky bit（`/tmp`、`passwd`）與安全風險 |
| 提權與 sudo | `su`、`sudo`、`sudoers`、`visudo`、最小授權、稽核紀錄 |
| POSIX ACL | `getfacl`／`setfacl`、何時該用 ACL 而不是多開群組 |
| Linux capabilities | 拆分 root 權力、`CAP_NET_BIND_SERVICE`、`getcap`／`setcap` |
| 核心的權限檢查 | real／effective／saved UID、`fork` 與 `exec` 的身分繼承 |
| 服務身分 | 專屬非 root 帳號、systemd `User=`、為什麼不該用 root 跑網路服務 |
| 登入與認證 | PAM、SSH 金鑰、`~/.ssh` 權限、`authorized_keys`、停用密碼 |
| 權限問題排查 | `Permission denied` 檢查順序：`namei -l`、`id`、`getfacl`、`strace` |
| 設計一份權限矩陣 | 共用目錄案例，組合群組、ACL、`sudo` 並驗證 |
| 容器裡的使用者 | 容器內外 UID 對應、non-root image（連到 O.6 與 DevOps 容器章節） |

### T.3 Shell 指令碼

| 標題 | 說明 |
| --- | --- |
| Shell 語法基礎 | 變數、條件、迴圈、函式 |
| 可重複執行的小腳本 |  |

### T.4 版本控制 Git

| 標題 | 說明 |
| --- | --- |
| Git 基本操作 | commit、branch、merge、rebase |
| 解 conflict 與讀歷史 |  |
| PR 流程與協作 |  |

### T.5 除錯與建置

| 標題 | 說明 |
| --- | --- |
| log 與除錯器 | 定位問題 |
| strace 與 tcpdump 入門 |  |
| 編輯器/IDE 與 Makefile |  |

## Z 綜合題

### Z.1 把它串起來

| 標題 | 說明 |
| --- | --- |
| 輸入網址按 Enter 之後 | 從鍵盤、DNS、TCP、TLS、HTTP 到伺服器回應，走一遍前面學過的層 |
