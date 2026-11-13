# AI 時代的程式碼審查：風險分級與合併閘門

對應文章：應用工程軌道 L4「AI 時代的程式碼審查：分層驗證與風險分級」。

- `review_gate/policy.py`：依改動的路徑、大小與作者，決定需要 0、1 或 2 位人類核准。
- `review_gate/gate.py`：先看自動化檢查與未處理的 finding，再數有效的人類核准。
- `tests/`：pytest 測試。
- `recognize/approval_gate.py`：文章「辨識」題用的 AI 產生程式碼，沒有測試，不被 pytest 掃描。

## 執行

```bash
uv run ruff check .
uv run pytest
```

不需要 GPU、網路或 API key；測試在一秒內跑完。
