---
title: AI 時代的程式碼審查：分層驗證與風險分級
description: 理解把審查拆成多層自動檢查、再依風險決定人工核准的模型，並為團隊設計合併閘門與導入步驟。
sidebar:
  order: 5
---

## 先想想

手冊問答助理的團隊導入程式碼代理人後，每週合併的 PR 從 20 個變成 60 個。資深工程師 Mei 還是逐行讀每一個 PR，現在每天花五個小時審查，PR 平均要等她三天。

團隊有人提議：「乾脆不要逐行讀了。」

不逐行讀的話，品質要靠什麼擔保？哪些 PR 仍然一定要有人看？

<details>
<summary>提示</summary>

先別問「AI 能不能取代人」。問「每一種風險，由哪一道檢查負責攔截」。

</details>

## 是什麼

**分層驗證（layered verification）** 是把程式碼審查從「合併前由人讀完整份 diff」的單一關卡，拆成多道各自攔截不同風險的檢查，再依改動的風險決定要不要加上人類核准。

它和你已經熟悉的 CI 最接近：CI 把測試分成 lint、型別、單元、整合幾層，分層驗證則在前面多加 AI 審查，在後面多加分階段發布與回滾。本文的模型整理自 Anthropic 的 Claude Code 團隊在 2026 年 10 月發表的[文章](https://claude.dev/blog/solving-code-review-at-anthropic/)，以及 Jason Stanley 對它的[評論](https://jasonstanley.substack.com/p/anthropics-code-review-as-a-case)；文中的數字查證於 2026-11-13，都是該團隊自報的相對值。

## 為什麼存在

傳統的做法是一位資深工程師花約 20 分鐘逐行讀完一個 PR。Stanley 指出，這個預算不是從第一原理算出來的，而是隱含地依「人寫的程式碼」的量與錯誤型態校準出來的習慣。

Anthropic 的 Claude Code 團隊約 90% 的程式碼由 Claude 寫。同一篇文章提到，他們的 monorepo 每天合併的變更，從 5 月的約 2,500 個成長到 9 月的 30,000 個以上。量變成十幾倍，逐行讀的人不會跟著變成十幾倍。

沒有重新設計的結果只有兩種：審查變成橡皮圖章，或資深工程師變成瓶頸。把時間從審查挪到「修被放行的壞程式碼」也沒有賺到，Stanley 稱之為 "human effort was moved not saved"。

所以該團隊把目標改寫成：用最少的人類審查時間，且不讓 revert 與 incident 變多。結果（自報）是：6 月到 9 月合併的 PR 增加一倍以上，被 revert 的比例下降 40% 以上。文章沒有給絕對的 revert 數字，讀的時候要把它當成方向，不當成保證。

## 怎麼運作

### 七層，不是一道關

```text
改動 ─▶ ① PR 前驗證     作者自己跑測試、lint，子代理人在乾淨的上下文審 diff
     ─▶ ② AI 審查       多個代理人找 bug，驗證代理人投票，過關的才留言
     ─▶ ③ 安全審查      獨立一輪；未處理的 finding 會擋合併
     ─▶ ④ CI            lint、型別、測試、建置，全數通過
     ─▶ ⑤ 人工核准      依風險分級：0、1 或 2 人
     ─▶ ⑥ 合併佇列與分階段發布   在最新 main 上重跑檢查；功能旗標先開給內部
     ─▶ ⑦ 偵測與回滾    關旗標或 revert，不需重新部署
```

①到⑤擋在合併之前，⑥與⑦假設一定有東西漏過去，目標是縮小影響範圍（blast radius）並快速修復。每一層只需要擋住一類風險，不需要完美。

第②層的細節值得抄：多個代理人平行找 bug，另一組驗證代理人對每個 finding 投票，只有共識夠高的才發出來，風格與清理類的意見直接丟掉。留下的 finding 標成 Important、Nit 或 Pre-existing，而且每一個都必須「修掉，或寫理由後標為已處理」。

### 風險分級：改動決定層級，不是作者

| 風險 | 例子 | 人類核准 |
| --- | --- | --- |
| 低 | 文件、文案、內部功能、有旗標保護的新增 | 0（自動檢查全綠即可） |
| 中 | 一般產品改動、敏感區域（快取、自動更新、system prompt） | 1 位 |
| 高 | 最高風險的程式碼、固定要兩人看的路徑 | 2 位 |

這個表背後有四條規則：

- 改動決定層級，作者只決定誰能走輕的那條路。只有受信任的作者才能走低風險路徑。
- 機器人的核准不算人類的核准。自動審查可以放行低風險改動，但補不了一個要求人類的名額。
- 核准綁定特定 commit，推了新 commit 就要重新核准。
- 無論誰打的字，每個上線的改動都有一個具名的人負責；請代理人寫這個 PR 的人，就是那個人。

套到手冊問答助理，路徑對應如下。

```python title="policy.py"
HIGH_PATHS = (".github/workflows/*", "deploy/*", "app/auth/*")
MEDIUM_PATHS = ("prompts/*", "app/*", "evals/*")
LOW_PATHS = ("docs/*", "*.md", "tests/*")

def tier_of_path(path: str) -> Tier:
    if any(fnmatch(path, p) for p in HIGH_PATHS):
        return Tier.HIGH
    if any(fnmatch(path, p) for p in MEDIUM_PATHS):
        return Tier.MEDIUM
    if any(fnmatch(path, p) for p in LOW_PATHS):
        return Tier.LOW
    # A path nobody classified is a path nobody thought about: ask a human.
    return Tier.MEDIUM

def classify(change: Change, trusted_authors: frozenset[str], big_change_lines: int = 400) -> Tier:
    """The change sets the tier. The author only decides who may use the lighter path."""
    tier = max((tier_of_path(f.path) for f in change.files), default=Tier.MEDIUM)
    if change.lines > big_change_lines:
        tier = max(tier, Tier.MEDIUM)
    if change.author not in trusted_authors:
        tier = max(tier, Tier.MEDIUM)
    return tier
```

注意 `prompts/system.md` 同時符合 `*.md` 與 `prompts/*`，取最高層級，所以是中風險。`evals/*` 也列為中風險：評估資料集是「這個系統應該做到什麼」的紀錄，改它等於改驗收標準。

合併閘門則把核准數得很嚴：

```python title="gate.py"
def valid_approvers(pr: PullRequest) -> set[str]:
    """People whose approval counts: a human, not the author, on the current head commit."""
    return {
        a.reviewer
        for a in pr.approvals
        if a.commit == pr.head_commit and not a.is_bot and a.reviewer != pr.change.author
    }
```

### 收斂：不讓審查無限跑下去

作者代理人與審查代理人都沒有「夠了」的概念，所以要寫進規則：三輪之後，審查者請作者重新考慮整個做法；第一輪之後只審新增的 diff；標出一類 bug，而不是一行一行指；同一個 finding 不重複報。

審查者本身也要改進。每次 incident 或 revert，都經專家確認後變成一個評估案例；審查者每天跑完整個評估集，prompt、工具或模型的改動只有在分數上升時才保留。這和 L3 的評估驅動開發是同一套做法，只是被評估的對象換成審查者。

### 人看什麼

人不再讀每一行，而是看這幾件事：該不該做這件事（方向、取捨、範圍）；最高風險的程式碼；會被別人照抄的架構模式；以及證據，也就是測試輸出、執行過的指令、審查者的 finding。作者必須答得出「還有什麼是現有檢查抓不到的」；答不出來，就該要求真正的人工審查，而不是快速按核准。

### 給團隊的導入清單

1. **先寫下目標與指標。** 每個 PR 的人類核准數（中位數與 P95）、人類留言數、等人的時間，護欄是 revert 率與 incident 率。先量 2 到 4 週的基準。
2. **補齊前面幾層。** CI 必須全過；PR 模板問「怎麼測試的」與「怎麼還原」。
3. **從路徑開始分級。** 用 CODEOWNERS 或像上面那樣的路徑表；沒分類的路徑一律當中風險；一開始全部都要人，確認指標穩定後再放寬低風險。
4. **核准綁 commit，機器人不算人。** 這兩條用測試鎖住。
5. **加斷路器（circuit breaker）。** 近期 revert 率超過基準的 1.5 倍，就把低風險升成中風險。這是本文的示意設計；Anthropic 的文章承認他們還沒有這類機制，並表示在更多程式碼走向零人工合併之前會想要。
6. **替「意圖」找一個負責人。** Stanley 指出，這條流水線裡沒有任何一層會對照組織的意圖來檢查 PR。評估集、規格、方向討論要有人擁有。

個別工程師該做的更簡單：把 PR 切小；附上測試輸出與還原方法；對自己（或自己的代理人）送出的每個改動負責。

回到一開始的問題：不逐行讀，品質由七層各自負責的檢查擔保，而不是一個人的眼睛。Mei 仍然要看的，是高風險路徑、會被抄的架構、以及「該不該做」；文件與內部低風險改動在自動檢查全綠後不用等她。要不要真的放手，由 revert 與 incident 這兩個護欄指標決定，不靠感覺。

## 取捨與替代方案

| 方案 | 優點 | 代價 | 適用情境 |
| --- | --- | --- | --- |
| 全部人工逐行讀 | 最保守，順便建立團隊共同脈絡 | 量大時成為瓶頸，疲勞後變橡皮圖章 | 小團隊、低流量、高風險領域 |
| AI 留言，人仍核准每個 PR | 找 bug 更快 | 人的時間仍隨 PR 數線性成長 | 剛導入、還沒有指標 |
| 風險分級（本文） | 人力集中在高風險 | 要維護分類規則與指標，要能快速回滾 | PR 量大，且有 CI 與功能旗標 |
| 低風險零人工合併 | 最快 | 漏過的錯只能靠偵測與回滾；斷路器尚未成熟 | 可逆、內部、自動化檢查很強的改動 |

什麼時候不該用：

- **回滾慢或不可逆。** 資料庫 schema 變更、對外 API、金流。分級把安全網放在「出事能快速撤回」，撤不回就該回到人工審查。
- **沒有可靠的測試與 CI。** AI 審查補的是 CI 之外的缺口，不是替代品。測試不可信時，後面每一層都建立在沙上。
- **沒有 revert 與 incident 指標。** 沒有護欄，你分不出人的時間是「省下來」還是「挪到事後修復」。
- **法規或合約要求人工簽核。** 這類要求優先於效率。

## 常見坑與故障案例

### 修 nit 的無限迴圈

症狀：一個 PR 過夜跑了數十輪審查，沒有收斂。Anthropic 的文章提到，一個 CLI PR 在作者喊停前跑了超過 50 輪（自報）。

原因：作者代理人修一個 nit，產生新 commit，觸發新一輪審查，審查者又找到新的 nit。文章還提到，審查者在後面幾輪標出的問題，約 59% 其實在第一輪就已存在於程式碼中。

對策：加上輪數上限、第一輪後只審新增的 diff、標出一類 bug 而非逐行、不重複報告，並把風格類意見直接丟掉。

### 核准算錯：舊 commit 與機器人

症狀：高風險的 workflow 檔案在核准後又被改過，PR 仍然合併了；或者只有審查機器人加一位人類，就滿足了兩人的要求。（示意案例）

原因：閘門把核准綁在 PR 上而不是 commit 上，或者單純數核准筆數，沒有區分人與機器人、也沒有去除重複的人。

對策：只計算 `commit == head_commit`、非機器人、非作者、去重後的核准；用測試鎖住這幾條。

### 只量「人省了多少時間」

症狀：PR 等待時間下降，兩個月後 revert 變多，on-call 變忙。（示意案例）

原因：人類時間從審查挪到了事後修復，帳面上只看到前半。

對策：把 revert 與 incident 當成護欄指標，和等待時間一起看；任何一邊惡化就收緊分級。

### 每一層都綠，但功能不該做

症狀：測試全過、審查全過、沒有人反對，上線後才發現這個功能和產品方向不符。（示意案例）

原因：自動化檢查的是程式碼，不是「這是不是組織要的東西」。

對策：把方向討論放在 PR 之前（issue 或規格）；保留一類人工決策：該不該做、要不要改範圍、要不要直接關掉這個 PR。

## 小練習

### 動手寫：斷路器

在範例的 `review_gate/policy.py` 加一個函式 `escalate(tier, recent_revert_rate, baseline, factor=1.5)`：當 `recent_revert_rate` 超過 `baseline * factor` 時，低風險改動要升成中風險；任何情況下都不能把層級往下調。

起始程式碼在 `playground/python/ai/engineering/l4/4-5-architecture-governance/ai-code-review/`。在 `tests/test_breaker.py` 寫三個測試：平靜時維持原層級、revert 暴增時 LOW 升為 MEDIUM、HIGH 不會被降級。

驗證做對了：`uv run pytest` 全過；然後把 `max(tier, Tier.MEDIUM)` 改成直接回傳 `Tier.MEDIUM`，HIGH 的那個測試應該變紅。

<details>
<summary>參考解答</summary>

```python title="policy.py"
def escalate(tier: Tier, recent_revert_rate: float, baseline: float, factor: float = 1.5) -> Tier:
    """Circuit breaker: when reverts climb past the baseline, a LOW change needs a human."""
    if recent_revert_rate > baseline * factor:
        return max(tier, Tier.MEDIUM)
    return tier
```

```python title="test_breaker.py"
def test_calm_period_keeps_the_tier() -> None:
    assert escalate(Tier.LOW, recent_revert_rate=0.02, baseline=0.02) == Tier.LOW


def test_revert_spike_lifts_low_to_medium() -> None:
    assert escalate(Tier.LOW, recent_revert_rate=0.05, baseline=0.02) == Tier.MEDIUM


def test_breaker_never_lowers_a_tier() -> None:
    assert escalate(Tier.HIGH, recent_revert_rate=0.05, baseline=0.02) == Tier.HIGH
```

關鍵是 `max`：斷路器只能收緊，不能放鬆。如果寫成直接回傳 `Tier.MEDIUM`，高風險的改動在 revert 暴增時反而只需要一位核准，第三個測試會失敗。

</details>

### 辨識

下面是 AI 產生的合併閘門，用在手冊問答助理的儲存庫。它通過了所有「正常路徑」的測試：文件改動 0 人、一般改動 1 人、workflow 改動 2 人。請做 code review，找出會讓不該合併的 PR 通過的地方。

```python title="approval_gate.py"
from dataclasses import dataclass

LOW_PREFIXES = ("docs/", "tests/")
HIGH_PREFIXES = (".github/workflows/", "deploy/")


@dataclass(frozen=True)
class Approval:
    reviewer: str
    commit: str
    is_bot: bool = False


def required_approvals(paths: list[str]) -> int:
    if any(p.startswith(HIGH_PREFIXES) for p in paths):
        return 2
    if all(p.startswith(LOW_PREFIXES) for p in paths):
        return 0
    return 1


def can_merge(
    paths: list[str],
    author: str,
    head_commit: str,
    approvals: list[Approval],
    checks_green: bool,
) -> bool:
    if not checks_green:
        return False
    given = [a for a in approvals if a.reviewer != author]
    return len(given) >= required_approvals(paths)
```

<details>
<summary>參考解答</summary>

三個問題都在 `can_merge` 的 `given` 那一行：

1. **沒有檢查 commit。** `head_commit` 傳進來卻沒被使用，任何 commit 上的核准都算數。作者先拿到對無害版本的核准，再推一個改 workflow 的 commit，PR 照樣通過。要加上 `a.commit == head_commit`。
2. **機器人的核准算成人類的核准。** `is_bot` 沒被檢查，審查機器人可以填滿人類名額。要加上 `not a.is_bot`。
3. **數的是筆數，不是人數。** 同一位審查者在高風險路徑核准兩次，就滿足「兩人」。要改成對 `reviewer` 去重的集合。

修正後：

```python
given = {
    a.reviewer
    for a in approvals
    if a.commit == head_commit and not a.is_bot and a.reviewer != author
}
return len(given) >= required_approvals(paths)
```

這三個錯誤都會讓正常路徑的測試維持綠燈，只有刻意構造「舊 commit」「機器人」「同一人兩次」的案例才會抓到，所以它們正是「看起來合理、測試會過」的那一類。

</details>

## 快閃卡

<details>
<summary>分層驗證和傳統的 PR 審查，差在哪裡？</summary>

傳統審查是合併前由人讀完整份 diff 的單一關卡。分層驗證把檢查拆成多層（PR 前驗證、AI 審查、安全審查、CI、風險分級人工核准、分階段發布、回滾），每層只攔一類風險，並假設會有東西漏過去，所以加上回滾來縮小影響範圍。

</details>

<details>
<summary>Anthropic 的 Claude Code 團隊把審查目標設成什麼？為什麼不是「讓 AI 全部取代人」？</summary>

目標是用最少的人類審查時間，且不讓 revert 與 incident 變多。因為如果人的時間只是從審查挪到事後修復，就沒有真正省下來，所以 revert 與 incident 是護欄。

</details>

<details>
<summary>PR 只改 .github/workflows/release.yml，作者是受信任成員，所有自動檢查全綠。依本文範例政策要幾位人類核准？</summary>

2 位。workflow 檔案在高風險路徑，層級由改動決定，不因為作者受信任或檢查全綠而降低。

</details>

<details>
<summary>PR 只改 docs/intro.md，所有檢查全綠，但作者不在受信任名單。要幾位人類核准？</summary>

1 位。只有受信任的作者才能走低風險路徑，不受信任的作者最低是中風險。

</details>

<details>
<summary>bob 核准了 commit a1b2c3d；作者之後推了 commit e5f6a7b，改了 prompts/system.md。bob 的核准還有效嗎？</summary>

無效。核准綁定特定 commit，新 commit 是系統無法擔保的內容，需要重新核准。

</details>

<details>
<summary>為什麼沒被任何規則分類到的路徑要預設成中風險？</summary>

沒被分類的路徑代表還沒有人想過它的風險。預設放行等於把未知當成安全；預設要人看，才是「失敗時往安全那一側」。

</details>

<details>
<summary>一個 PR 過夜跑了五十輪審查仍未收斂。可能的原因與對策？</summary>

作者代理人修 nit 又觸發新一輪審查，缺少收斂規則。對策：輪數上限、第一輪後只審新增的 diff、標出一類 bug 而不是逐行、不重複報告、丟掉風格類意見。

</details>

<details>
<summary>風險分級和「全部人工逐行讀」比，什麼情況下前者反而不適合？</summary>

回滾慢或不可逆（資料庫 schema 變更、對外 API）、沒有可靠的 CI、沒有 revert 與 incident 指標，或法規要求人工簽核時。分級的安全網是快速回滾與指標，缺了它們就要回到人工審查。

</details>

## 延伸閱讀

- [How the Claude Code team does code review](https://claude.dev/blog/solving-code-review-at-anthropic/)：本文的主要來源，描述七層模型、風險分級與收斂規則；數字是自報的相對值
- [Anthropic's code review as a case](https://jasonstanley.substack.com/p/anthropics-code-review-as-a-case)：Jason Stanley 的評論，把這件事看成「工作治理」的重寫，並指出意圖檢查的缺口
- [Google Engineering Practices: Code Review](https://google.github.io/eng-practices/review/)：人類審查者該看什麼、該怎麼回饋，是分層驗證裡「人工核准」那一層的基本功
- [DORA](https://dora.dev/)：軟體交付績效的研究，可以拿來對照你自己的 revert 與 incident 指標
