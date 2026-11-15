---
title: 工具與效率目錄
description: 工具與效率領域所有章節一覽，包含尚未寫好的章節。
sidebar:
  order: 0
---

<!-- 由 scripts/gen-toc.mjs 從 docs/contents/ 產生，請勿手動修改；改目錄請改 docs/contents/ 後重跑 `pnpm toc`。 -->

下列為工具與效率領域規劃的所有章節。左側欄只列出已寫好的章節，其餘陸續補上。

## E 編輯器與 IDE

### E.1 導覽與搜尋

- 專案索引原理：跳到定義、找用法、全域搜尋
- 快捷鍵與命令面板的學習方法

### E.2 重構與程式碼品質

- 安全重構：改名、抽取函式、移動符號
- 格式化、Lint 與保存時自動修正

### E.3 除錯器

- 中斷點、條件中斷點、監看運算式
- 遠端除錯與除錯容器內的程式

### E.4 擴充與設定管理

- 外掛的選擇與風險
- 設定同步與團隊共用設定

## K 終端機與 Shell 環境

### K.1 終端機工作流

- 多工：tmux 與分頁、分割視窗
- 歷史搜尋、別名與函式

### K.2 Shell 設定

- dotfiles 管理與可重現的開發環境
- 版本管理工具：在同一台機器切換語言版本

## G Git 工具

### G.1 閱讀歷史

- 圖形介面與 `git log` 的搭配
- blame、bisect 與找出引入問題的 commit

### G.2 整理與修復

- 互動式 rebase：整理 commit
- reflog 與救回誤刪的工作
- 衝突解決工具與合併策略

## B 資料庫用戶端

### B.1 連線與查詢

- [DBeaver 功能導覽：後端工程師該學哪些](/tools/b-db-client/b-1-connect-query/dbeaver-overview/)
- 連線設定、SSH tunnel 與連線安全
- 查詢編輯器、結果匯出與匯入

### B.2 看懂資料庫在做什麼

- 在用戶端讀執行計畫
- 查看鎖、連線與慢查詢
- 在正式環境安全操作：唯讀帳號、交易與確認

## H API 與網路工具

### H.1 發請求

- 圖形化 API 工具與 curl 的取捨
- 環境變數、集合與請求鏈

### H.2 看見網路

- 瀏覽器開發者工具的 Network 面板
- 用代理工具觀察與重現請求

## D 除錯與效能分析

### D.1 定位錯誤

- 從日誌、堆疊追蹤到重現步驟
- 用除錯器逐步縮小問題範圍

### D.2 找出瓶頸

- 用 profiler 看 CPU 與記憶體（以 Go pprof 為例）
- 火焰圖怎麼讀

## A AI Agent 輔助開發

### A.1 任務與記憶管理

- [Agent 任務與記憶管理](/tools/a-agent-dev/a-1-task-memory/beads-task-memory/)

### A.2 多 Agent 並行與監控

- [Herdr 多 Agent 並行與監控](/tools/a-agent-dev/a-2-multi-agent/herdr-multi-agent/)
