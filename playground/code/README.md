# 程式語言 100 題學習路線

目標：用 100 題從 L1 語法概念練到 L4 實戰應用。先用 Go 建立題庫與 pseudocode，再把同一套題目延伸到其他語言。

狀態：規劃完成，Phase 1（Go）進行中。

## 目標熟練度

| 語言 | 用途 | 目標 | 題數 |
|---|---|---|---|
| Go | 後端微服務（工作主力） | L4 | 100 |
| Java | 後端商業邏輯 | L4 | 100 |
| Python | AI | L3 | 約 65 |
| TypeScript | 前端 | L2 | 約 30 |
| C++ | 硬體與底層 | L2 | 約 30 |

## 設計原則

1. **一次只主修一門語言。** 同時練五門效果最差。前一門語言進入維護模式，只在新語言做對照時回頭看。
2. **共用題寫一次，所有語言都用。** L1、L2 與 L3 前 10 題是共用題，`exercise.md` 和 `pseudocode.md` 與語言無關。
3. **L3–L4 依語言分流。** 每門語言的核心能力不同，同一套題目涵蓋不到，所以這部分是語言專屬題。
4. **pseudocode 保持語言中立。** 不要寫成 Go 的風格，例如不要預設有 goroutine 或多回傳值。
5. **每題都要有對照筆記。** 第二門語言開始，在 `notes.md` 記錄這題在各語言的差異。這是跨語言學習最有價值的產出。
6. **每題都要有測試。** 從 L2 開始，每題的實作都附上單元測試。

## 學習路線

| Phase | 語言 | 範圍 | 重點 |
|---|---|---|---|
| 1 | Go | 全部 100 題 | 建立題庫與 pseudocode，這是所有語言的模板 |
| 2 | Java | 共用 60 題 + 專屬 40 題 | 共用題快速重做，逐題對照 Go |
| 3 | Python | L1–L2 共用 50 題 + L3 專屬 15 題 | 換語法不換邏輯，專屬題偏 AI |
| 4 | TypeScript | 從 L1–L2 挑約 30 題 | union、type narrowing、`async`/`await` |
| 5 | C++ | 從 L1–L2 挑約 30 題 | pointer、reference、RAII、`const`、STL |

Java 排在 Python 前面的理由：Go 和 Java 都是 L4 目標，也都是後端。用同一類問題連續對照，學習效果最好。

TypeScript 和 C++ 依序學，不要同時學。工作上先需要哪一門，就先學那一門。

### 進階門檻

- **進入下一個 Level**：不查文件能寫完該 Level 八成的題目，而且每題都有測試。
- **進入下一個 Phase**：前一門語言的 L3 已經完成。

## 題目規劃

| Level | 題號 | 題數 | 類型 | 主題 |
|---|---|---|---|---|
| L1 語法概念 | 001–025 | 25 | 共用 | 型別、控制流、函式、錯誤處理、集合 |
| L2 資料處理 | 026–050 | 25 | 共用 | 字串、map/list、排序、資料建模、檔案 I/O、JSON |
| L3 慣用寫法 | 051–060 | 10 | 共用 | HTTP、CLI、設定檔、測試 |
| L3 慣用寫法 | 061–075 | 15 | 專屬 | 依語言而定 |
| L4 實戰 | 076–100 | 25 | 專屬 | 依語言而定 |

### L1 語法概念（共用，001–025）

| 章節 | 題號 | 內容 |
|---|---|---|
| A 型別與變數 | 001–005 | 基本型別、常數、型別轉換、格式化輸出 |
| B 控制流 | 006–010 | `if`、`switch`、迴圈、提早 return |
| C 函式 | 011–015 | 參數、回傳值、多回傳值或 tuple、遞迴 |
| D 錯誤處理 | 016–020 | 輸入驗證、錯誤回傳與例外、錯誤包裝 |
| E 集合 | 021–025 | 陣列、list、map 的增刪查改與走訪 |

| 題號 | 題目 | 重點 |
|---|---|---|
| 001 | BMI Measurement | 常數、函式、分類、格式化輸出 |
| 002 | Temperature Conversion | 整數與浮點數除法、對照表輸出 |
| 003 | Seconds to Duration | `/` 與 `%`、單位換算 |
| 004 | Receipt Formatting | 金額用整數、四捨五入、欄寬對齊 |
| 005 | String Number Conversion | 字串轉數字、截斷與取整方向 |
| 006 | Grade Letter | 提早 return、`switch` 範圍判斷 |
| 007 | FizzBuzz Extended | 迴圈、`%`、規則資料化 |
| 008 | Days in Month | 閏年、`switch` 多值 `case` |
| 009 | Multiplication Table | 巢狀迴圈 |
| 010 | Prime Numbers | `break`、`continue` |
| 011 | Discount Calculator | 函式拆分、純函式 |
| 012 | Multiple Returns | 多回傳值或 tuple、負數除法 |
| 013 | Value vs Reference | 傳值與傳參考 |
| 014 | Recursion | 遞迴與迴圈、整數溢位 |
| 015 | Function Values | 高階函式、closure |
| 016 | Input Validation | 驗證與錯誤訊息 |
| 017 | Safe Divide | 錯誤值與例外、`Inf` 與 `NaN` |
| 018 | Error Kinds | 依錯誤種類分支 |
| 019 | Error Wrapping | 錯誤鏈、判斷根本原因 |
| 020 | Custom Error and Cleanup | 自訂錯誤型別、`defer` / `finally` |
| 021 | Array Basics | 固定長度陣列、原地反轉 |
| 022 | Todo List | 動態清單、邊界檢查 |
| 023 | Phone Book | map 增刪查改、走訪順序 |
| 024 | Matrix | 巢狀集合 |
| 025 | Inventory | L1 綜合題 |

### L2 資料處理（共用，026–050）

| 章節 | 題號 | 內容 |
|---|---|---|
| A 字串 | 026–030 | 切割、搜尋、取代、統計字元 |
| B map 與 list | 031–035 | 分組、計數、去重、合併 |
| C 排序與搜尋 | 036–040 | 自訂排序、二分搜尋、Top-K |
| D 資料建模 | 041–045 | struct 或 class、方法、介面 |
| E 檔案與 JSON | 046–050 | 讀寫檔案、CSV、JSON 序列化 |

### L3 慣用寫法（共用，051–060）

| 章節 | 題號 | 內容 |
|---|---|---|
| A HTTP | 051–053 | HTTP server、cookie 與 session、HTTP client |
| B CLI | 054–055 | 解析參數、讀 stdin、exit code |
| C 設定 | 056–057 | 讀設定檔與環境變數 |
| D 測試 | 058–060 | 單元測試、mock、測試覆蓋率 |

### 語言專屬題（061–100）

**Go**

| 章節 | 題號 | 內容 |
|---|---|---|
| L3 並行 | 061–065 | goroutine、channel、`select`、worker pool |
| L3 `context` | 066–068 | 取消、逾時、傳遞 request 範圍的值 |
| L3 interface | 069–071 | interface 組合、泛型 |
| L3 測試 | 072–075 | table-driven test、benchmark、race detector |
| L4 REST 服務 | 076–082 | router、middleware、驗證、錯誤回應格式 |
| L4 微服務通訊 | 083–088 | gRPC、protobuf、服務間呼叫 |
| L4 資料庫 | 089–094 | SQL、交易、migration |
| L4 上線 | 095–100 | graceful shutdown、logging、metrics、Docker |

**Java**

| 章節 | 題號 | 內容 |
|---|---|---|
| L3 OOP | 061–065 | 繼承、抽象類別、介面、多型 |
| L3 泛型與集合 | 066–068 | 泛型、Collections framework |
| L3 Stream | 069–071 | lambda、Stream API、Optional |
| L3 例外與測試 | 072–075 | 例外階層、JUnit、Mockito |
| L4 Spring Boot | 076–082 | REST API、依賴注入、驗證 |
| L4 資料存取 | 083–088 | JPA、交易管理 |
| L4 商業邏輯 | 089–094 | 訂單與付款的狀態機 |
| L4 安全與整合 | 095–100 | Spring Security、整合測試 |

**Python**（只到 L3）

| 章節 | 題號 | 內容 |
|---|---|---|
| L3 型別 | 061–063 | type hints、dataclass |
| L3 迭代 | 064–066 | generator、iterator |
| L3 非同步 | 067–069 | `asyncio` |
| L3 資料 | 070–072 | NumPy、Pandas |
| L3 AI | 073–075 | 呼叫 LLM API、簡易 RAG |

**TypeScript、C++**（只到 L2）

從 L1–L2 挑約 30 題，優先選能凸顯語言特性的題目：

- TypeScript：D 資料建模（union、type narrowing）、E 檔案與 JSON（`async`/`await`）
- C++：C 函式（pointer、reference）、D 資料建模（RAII、`const`）、E 集合（STL 容器）

## 資料夾架構

```
playground/code/
├── README.md
├── core/                         # 共用題（L1、L2、L3 前 10 題）
│   ├── L1/
│   │   └── 001-bmi-measurement/
│   │       ├── exercise.md       # 題目與需求
│   │       ├── pseudocode.md     # 語言中立的 pseudocode
│   │       ├── notes.md          # 跨語言對照筆記
│   │       ├── golang/
│   │       ├── java/
│   │       └── ...               # 輪到該 Phase 才新增
│   ├── L2/
│   └── L3/
│       └── 051-cookie-session/
├── golang/                       # Go 專屬題
│   ├── L3/
│   └── L4/
├── java/                         # Java 專屬題
│   ├── L3/
│   └── L4/
├── python/                       # Python 專屬題
│   └── L3/
└── commands/                     # 獨立的 Linux/Bash 訓練營
```

### 命名規則

- 題目資料夾：`<三位數題號>-<kebab-case 題名>`，例如 `001-bmi-measurement`。
- 語言資料夾：`golang`、`java`、`python`、`typescript`、`cpp`。
- 專屬題放在 `<語言>/<Level>/<題號>-<題名>/`，結構與共用題相同，但沒有其他語言的資料夾。

### 新增一題

1. 建立題目資料夾，寫 `exercise.md`。
2. 寫 `pseudocode.md`，確認沒有用到特定語言的功能。
3. 新增目前 Phase 的語言資料夾並實作，附上測試。
4. 第二門語言起，在 `notes.md` 補上對照。

## 進度

| Level | Go | Java | Python | TypeScript | C++ |
|---|---|---|---|---|---|
| L1 | 1/25 | 1/25 | 1/25 | 1/25 | 1/25 |
| L2 | 0/25 | 0/25 | 0/25 | 0/25 | 0/25 |
| L3 | 1/25 | 1/25 | 1/15 | — | — |
| L4 | 0/25 | 0/25 | — | — | — |

001 和 051 是規劃前完成的題目，已經有五種語言的實作，保留作為參考。之後依 Phase 順序新增語言。
