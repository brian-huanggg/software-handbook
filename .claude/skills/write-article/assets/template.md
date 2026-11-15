---
title: <與 docs/contents/<field>.md 項目文字一致>
description: <讀完能做到什麼，60 字以內>
sidebar:
  order: <序號>
---

<!--
  How to use: copy this file, fill in every section, and delete all comments when done.
  The nine ## headings are literal Chinese strings: do not change a single character, and keep the order.
  Frontmatter placeholders: title = same as the item text in docs/contents/<field>.md;
  description = what the reader can do after reading, within 60 characters; order = sequence number in the section.
-->

## 先想想

<!--
  Think First.
  One concrete situation (preferably one the URL shortener service runs into), plus one question, so the reader reasons it out first.
  3 to 6 sentences. Do not give the answer here; answer it explicitly at the end of 「怎麼運作」.
  You may give one hint with <details><summary>提示</summary>…</details>.
-->

## 是什麼

<!--
  What It Is.
  The first paragraph is a one-sentence definition, with the term itself in bold.
  Then at most one paragraph of supplement: what kind of thing it is, and which concept the reader already knows it is closest to.
-->

## 為什麼存在

<!--
  Why It Exists.
  Which pain point it solves. First say what happens "without it" (preferably with numbers or concrete consequences), then how it changes the situation.
-->

## 怎麼運作

<!--
  How It Works.
  Principles and diagrams. Go from shallow to deep, using ### subheadings for the steps.
  At least one figure (format in "Figures" of references/style-guide.md).
  End with a paragraph that answers the question from 「先想想」, starting with 「回到一開始的問題：」.
-->

## 取捨與替代方案

<!--
  Trade-offs and Alternatives.
  What the cost is, and when not to use it. At least one comparison table: | 方案 | 優點 | 代價 | 適用情境 |
  The last paragraph is 「什麼時候不該用」, with at least two points.
-->

## 常見坑與故障案例

<!--
  Common Pitfalls and Failure Cases.
  3 to 5 pitfalls, each with a ### heading, whose content is in order: symptom → cause → how to avoid or fix.
  At least one is a real incident (with a link to a public incident report or article); if no reliable source can be found, label it 「示意案例」.
-->

## 小練習

<!--
  Exercises.
  1 to 3 exercises from easy to hard, with at least one hands-on and one recognition exercise (rules in "Exercise types in 「小練習」" of references/style-guide.md).
  Hands-on (in Go, Python for the `ai` field): state the goal, where the starter code is (playground/<field>/...), and how to verify it is done (for example which test will pass).
  Recognition: use ### 辨識 as the subheading, paste a piece of AI-generated code (code block title is only the file name), and ask the reader to do a code review: good vs. bad, or find the errors hidden in it.
  Put each exercise's reference answer in <details><summary>參考解答</summary>…</details>.
-->

## 快閃卡

<!--
  Flashcards.
  5 to 10 cards, covering the core concepts of 「是什麼」「為什麼存在」「怎麼運作」「取捨與替代方案」「常見坑」 of this article.
  Writing rules are in "Flashcards" of references/style-guide.md.
  One <details> per card: the front goes inside <summary> (plain text or HTML only, no Markdown),
  with one blank line between <summary> and the back, and one between the back and </details>. For example:

  <details>
  <summary>https://a.com 和 https://a.com:443 是同源嗎？</summary>

  是。https 的預設 port 是 443，明確寫出的預設 port 會被正規化，兩者的 scheme、host、port 都相同。

  </details>
-->

## 延伸閱讀

<!--
  Further Reading.
  3 to 6 links, preferring official documentation, original papers, RFCs, and well-known engineering blogs.
  Format: - [標題](URL)：一句話說明為什麼值得讀
  Include only links you are sure exist; if unsure, leave them out.
-->
