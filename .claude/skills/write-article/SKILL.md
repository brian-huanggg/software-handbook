---
name: write-article
description: 撰寫或改寫本書（後端、DevOps、資安、測試、AI）的一篇章節文章。當使用者要求「寫一篇」「寫這章」「write article/chapter」，或指定 docs/contents/<field>.md 或 docs/contents/ai/ 下某份軌道目錄裡的某個項目時使用。
---

# 撰寫一篇書籍文章

本書寫給自學的初級工程師，以 Go 為主要語言；`ai` 領域例外，主要語言是 Python。全書分成數個領域（field），每個領域一份目錄：`docs/contents/backend.md`、`docs/contents/devops.md`、`docs/contents/cyber.md`、`docs/contents/testing.md`。網站是 `site/`（Astro Starlight），每個領域有自己的側欄。

`ai` 領域分成三條軌道（track），每條軌道一份目錄：`docs/contents/ai/foundations.md`（理論基礎）、`docs/contents/ai/engineering.md`（應用工程）、`docs/contents/ai/ai-agents.md`（代理人）。三條軌道共用一個側欄 `site/src/sidebar/ai.ts`。下文寫 `<field>` 的地方，對 `ai` 領域一律讀作 `ai/<track>`：目錄是上述該軌道的目錄檔（代理人軌道的檔名是 `ai-agents.md`，避免被誤認為代理人的指示檔，但 `<track>` 仍是 `agents`），文章路徑是 `site/src/content/docs/ai/<track>/...`，範例路徑是 `examples/ai/<track>/...`。
每篇文章都必須遵守本 skill，讓不同 agent 寫出來的章節讀起來像同一位作者。

規則的優先順序：使用者這次的指示 > 本檔 > `style-guide.md` > 你自己的習慣。

## 輸入

開始前先確認這四件事；缺任何一項就問使用者，不要猜：

1. **領域**：`backend`、`devops`、`cyber`、`testing` 或 `ai`。`ai` 領域還要確認是哪一條軌道：`foundations`、`engineering` 或 `agents`。
2. **主題**：對應 `docs/contents/<field>.md` 的哪一個項目（例如「2.3 快取穿透、擊穿、雪崩與對策」）。
3. **層級**：L0–L4，由該項目所在的章節決定。層級決定深度與篇幅（見 `style-guide.md`）。
4. **是新寫還是改寫**：改寫時先完整讀過現有文章。

## 流程

1. **讀上下文**
   - `docs/contents/<field>.md`：找出本篇的前一篇與後一篇，知道讀者已經學過什麼、接下來要學什麼。
   - 同一節已寫好的文章（`site/src/content/docs/<field>/<level>/<section>/`），避免重複講解，並沿用相同的例子。
   - `docs/glossary.md`：術語一律照表使用。
   - `style-guide.md` 與 `template.md`（本 skill 目錄下）。
2. **寫大綱**：依 `template.md` 的八個段落，每段一到兩句話說明要寫什麼。主題橫跨兩篇以上，或篇幅明顯超出範圍時，先把大綱給使用者確認。
3. **寫正文**：複製 `template.md`，照段落順序填寫。段落標題一字不改，不新增、不刪除、不調換順序。
4. **寫程式範例**：放在 `examples/<field>/<level>/<section>/<slug>/`，是可獨立執行的 Go module；`ai` 領域則是獨立的 Python `uv` 專案（見 `style-guide.md` 的「程式碼」）。
5. **驗證**：逐項檢查 `checklist.md`。Go 範例在環境有 `go` 時執行 `go vet ./...` 與 `go test ./...`；Python 範例在環境有 `uv` 時執行 `uv run ruff check .` 與 `uv run pytest`。工具不存在、或範例需要 GPU、付費 API 而無法執行時，在回報中寫明「程式碼未經執行驗證」與原因。
6. **收尾**
   - 新術語加進 `docs/glossary.md`。
   - 在 `docs/contents/<field>.md` 把該項目的 `- [ ]` 改成 `- [x]`，並在後面加上文章連結。
   - 回報：檔案路徑、字數、未通過或略過的檢查項目、想請使用者決定的地方。

## 檔案位置與命名

```
site/src/content/docs/<field>/<level>/<section>/<slug>.md
examples/<field>/<level>/<section>/<slug>/
```

- `<field>`：`backend`、`devops`、`cyber`、`testing`，或 `ai/foundations`、`ai/engineering`、`ai/agents`。
- `<level>`：`l0` … `l4`
- `<section>`：**一律照 `site/src/sidebar/<field>.ts` 裡該節的 `dir`**（`ai` 領域照 `site/src/sidebar/ai.ts` 裡該軌道的 `dir`），例如 `2-3-cache`，不要自己取名。該節還不在檔案裡（新領域或新的一節）時，依「節號加英文 kebab-case」取名，並把 `{ label, dir }` 加進該檔對應層級的 `sections`，`label` 與目錄的節標題一致。
- 側欄只顯示已有文章的節，文章本身依 frontmatter 的 `sidebar.order` 排序，不必另外登記。
- `<slug>`：英文 kebab-case、2 到 5 個字，例如 `cache-stampede`。

Frontmatter 固定為：

```yaml
---
title: 快取穿透、擊穿、雪崩與對策     # 與 docs/contents/<field>.md 的項目文字一致
description: 一句話說明讀完能做到什麼    # 60 字以內
sidebar:
  order: 5                             # 本項目在該節中的序號（從 1 起算）
---
```

## 資安主題的額外規則

- 攻擊示範只針對讀者在本機自己寫、自己跑的程式（例如 `localhost` 上的範例服務），絕不針對第三方系統或真實網站。
- 每個攻擊都要在同一篇文章裡附上修補方式，並驗證修補後攻擊確實失效。
- 在「怎麼運作」段落開頭放一個 `:::caution` 提醒：只能在自己擁有或獲得授權的系統上測試。
- 不提供繞過偵測、隱藏蹤跡或大規模掃描的內容。
