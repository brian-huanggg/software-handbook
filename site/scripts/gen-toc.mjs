// Generates src/content/docs/<field>/toc.md from the outlines in
// ../docs/contents/. The outlines stay the single source of truth: this script
// copies their level/section headings and article tables, minus the 狀態
// column (status changes often and would drift from the site).
//
//   node scripts/gen-toc.mjs          write the toc pages
//   node scripts/gen-toc.mjs --check  exit 1 if any toc page is stale
import { readFileSync, writeFileSync, existsSync } from 'node:fs';

const outlines = new URL('../../docs/contents/', import.meta.url);
const docs = new URL('../src/content/docs/', import.meta.url);

// field -> outline sources. Several sources (the AI tracks) become one page,
// each under its own heading.
const fields = {
	cs: { title: '計算機科學', sources: [{ file: 'cs.md' }] },
	backend: { title: '後端', sources: [{ file: 'backend.md' }] },
	frontend: { title: '前端', sources: [{ file: 'frontend.md' }] },
	devops: { title: '維運', sources: [{ file: 'devops.md' }] },
	cyber: { title: '資安', sources: [{ file: 'cyber.md' }] },
	testing: { title: '測試與品質', sources: [{ file: 'testing.md' }] },
	ai: {
		title: 'AI',
		sources: [
			{ file: 'ai/foundations.md', track: '理論基礎' },
			{ file: 'ai/engineering.md', track: '應用工程' },
			{ file: 'ai/ai-agents.md', track: 'Agent' },
		],
	},
	tools: { title: '工具與效率', sources: [{ file: 'tools.md' }] },
};

// Drop pointers to outline files ("（`docs/contents/x.md`）"); they mean nothing on the site.
// Outline links point at article files ("../../site/src/content/docs/<path>.md");
// rewrite them to the page URL.
const clean = (s) =>
	s
		.replace(/\]\((?:\.\.\/)+site\/src\/content\/docs\/([^)]+?)\.mdx?\)/g, '](/$1/)')
		.replace(/[（(]`docs\/contents\/[^`]*`[）)]/g, '')
		.replace(/`docs\/contents\/[^`]*`/g, '');

// Cells of a table row, without the outer pipes.
const cells = (line) => line.trim().replace(/^\||\|$/g, '').split('|').map((c) => c.trim());

/** Body of one outline: from its first `##` heading on, minus 寫作與維護規則. */
function convert(text, shift) {
	const lines = text.split('\n');
	const out = [];
	let started = false;
	let skip = false;
	let dropCol = -1;
	for (let i = 0; i < lines.length; i++) {
		const line = lines[i];
		const h = /^(#{2,3}) /.exec(line);
		if (h) {
			started = true;
			skip = h[1] === '##' && /寫作與維護規則/.test(line);
			if (!skip) out.push('#'.repeat(h[1].length + shift) + line.slice(h[1].length));
			continue;
		}
		if (!started || skip) continue;
		if (line.startsWith('|')) {
			const c = cells(line);
			// A table is a status table iff its header row starts with 狀態.
			const isHeader = i > 0 && !lines[i - 1].startsWith('|');
			if (isHeader) dropCol = c[0] === '狀態' ? 0 : -1;
			const kept = dropCol === 0 ? c.slice(1) : c;
			out.push(clean(`| ${kept.join(' | ')} |`));
			continue;
		}
		// tools.md lists articles as "- [ ] title"; the checkbox is status.
		out.push(clean(line).replace(/^- \[[ x]\] /, '- '));
	}
	return out.join('\n').replace(/\n{3,}/g, '\n\n').trim();
}

const check = process.argv.includes('--check');
let stale = 0;
for (const [field, { title, sources }] of Object.entries(fields)) {
	const parts = sources.map(({ file, track }) => {
		const body = convert(readFileSync(new URL(file, outlines), 'utf8'), track ? 1 : 0);
		return track ? `## ${track}\n\n${body}` : body;
	});
	const page = `---
title: ${title}目錄
description: ${title}領域所有章節一覽，包含尚未寫好的章節。
sidebar:
  order: 0
---

<!-- 由 scripts/gen-toc.mjs 從 docs/contents/ 產生，請勿手動修改；改目錄請改 docs/contents/ 後重跑 \`pnpm toc\`。 -->

下列為${title}領域規劃的所有章節。左側欄只列出已寫好的章節，其餘陸續補上。

${parts.join('\n\n')}
`;
	const path = new URL(`${field}/toc.md`, docs);
	if (check) {
		if (!existsSync(path) || readFileSync(path, 'utf8') !== page) {
			console.error(`stale: ${field}/toc.md`);
			stale++;
		}
	} else {
		writeFileSync(path, page);
		console.log(`wrote ${field}/toc.md`);
	}
}
process.exit(stale ? 1 : 0);
