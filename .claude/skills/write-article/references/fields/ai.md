# `ai` 領域的額外規則

只在寫 `ai` 領域的文章時讀本檔。本檔與 `references/style-guide.md` 衝突時，以本檔為準。

## 讀者與寫法

- 主要語言是 Python，不是 Go。不從 Python 語法從零講起，但 NumPy、pandas、PyTorch 等工具在 L1 第一次出現時要講清楚。
- 理論基礎軌道的數學：先給直覺，再給公式，最後用 NumPy 或 PyTorch 驗證公式。公式用行內程式碼或 ` ```text ` 區塊寫出（本站沒有安裝數學渲染器），每個符號都要說明意思。
- 不假設讀者有 GPU 或付費 API。需要 GPU 的範例要附上 CPU 可跑完的縮小版設定；會產生 API 費用的練習要標出大約金額，並優先提供本機 Ollama 的做法。
- 模型名稱、價格、基準分數變得很快：寫明查證日期，不寫「目前最強」這類會過期的說法。
- 代理人與應用工程的安全主題（提示注入、資料外洩）同樣遵守 `SKILL.md` 的「資安主題的額外規則」：攻擊只針對自己的範例程式，並附上修補與驗證。

## 貫穿例子

取代全書的短網址服務，依軌道而不同，見各軌道目錄開頭：

- 理論基礎：表格資料、MNIST 與一份繁體中文語料
- 應用工程：「手冊問答助理」
- 代理人：「手冊維護代理人」

版本與層級相符（L1→v1……）。短網址服務只在需要連回後端領域時提到。

## Python 範例

- 每個範例放在 `playground/python/ai/<track>/<level>/<section>/<slug>/`，是獨立的 `uv` 專案：`pyproject.toml`（鎖定 Python 版本與套件版本）、`uv.lock`、原始碼與 `tests/`（pytest）。
- 程式碼通過 `ruff check`，公開函式加型別標註。
- 訓練類範例固定隨機種子，並在 README 寫明預期的執行時間與硬體。
- 需要 API key 時只從環境變數讀取，並附上 `.env.example`，不放真實金鑰。
- 片段標題寫法與 Go 相同：` ```python title="playground/python/ai/engineering/l2/2-4-rag/handbook-qa-v1/src/qa.py" `，省略處用 `# ...`。
- 其餘規則（片段長度、註解寫「為什麼」）照 `references/style-guide.md` 的「程式碼」。
