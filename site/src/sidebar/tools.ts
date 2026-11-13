import { fieldSidebar, type Level } from './helpers';

// Mirrors docs/contents/tools.md. Tools is grouped by tool category instead of
// L0–L4, so each category takes the place of a level. The `dir` values are the
// canonical directory names under src/content/docs/tools/; articles must use
// them. Labels drop the outline's letter codes; the codes stay in tools.md and
// in the `dir` values.
export const levels: Level[] = [
	{
		label: '編輯器與 IDE',
		dir: 'e-editor',
		sections: [
			{ label: '導覽與搜尋', dir: 'e-1-navigation' },
			{ label: '重構與程式碼品質', dir: 'e-2-refactoring' },
			{ label: '除錯器', dir: 'e-3-debugger' },
			{ label: '擴充與設定管理', dir: 'e-4-extensions' },
		],
	},
	{
		label: '終端機與 Shell 環境',
		dir: 'k-terminal',
		sections: [
			{ label: '終端機工作流', dir: 'k-1-terminal-workflow' },
			{ label: 'Shell 設定', dir: 'k-2-shell-config' },
		],
	},
	{
		label: 'Git 工具',
		dir: 'g-git',
		sections: [
			{ label: '閱讀歷史', dir: 'g-1-reading-history' },
			{ label: '整理與修復', dir: 'g-2-cleanup-recovery' },
		],
	},
	{
		label: '資料庫用戶端',
		dir: 'b-db-client',
		sections: [
			{ label: '連線與查詢', dir: 'b-1-connect-query' },
			{ label: '看懂資料庫在做什麼', dir: 'b-2-inspect' },
		],
	},
	{
		label: 'API 與網路工具',
		dir: 'h-http',
		sections: [
			{ label: '發請求', dir: 'h-1-requests' },
			{ label: '看見網路', dir: 'h-2-network-visibility' },
		],
	},
	{
		label: '除錯與效能分析',
		dir: 'd-debug-profile',
		sections: [
			{ label: '定位錯誤', dir: 'd-1-locating-bugs' },
			{ label: '找出瓶頸', dir: 'd-2-bottlenecks' },
		],
	},
];

export default fieldSidebar('tools', levels);
