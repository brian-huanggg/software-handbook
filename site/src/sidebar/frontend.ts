import { fieldSidebar, type Level } from './helpers';

// Mirrors docs/contents/frontend.md. The `dir` values are the canonical
// directory names under src/content/docs/frontend/; articles must use them.
export const levels: Level[] = [
	{
		label: 'L0 瀏覽器與 Web 基礎',
		dir: 'l0',
		sections: [
			{ label: '0.1 瀏覽器如何運作', dir: '0-1-browser' },
			{ label: '0.2 HTML 與 CSS', dir: '0-2-html-css' },
			{ label: '0.3 JavaScript 核心', dir: '0-3-javascript' },
		],
	},
	{
		label: 'L1 TypeScript',
		dir: 'l1',
		sections: [
			{ label: '1.1 型別系統基礎', dir: '1-1-type-basics' },
			{ label: '1.2 進階型別', dir: '1-2-advanced-types' },
			{ label: '1.3 工具鏈', dir: '1-3-toolchain' },
		],
	},
	{
		label: 'L2 React 核心',
		dir: 'l2',
		sections: [
			{ label: '2.1 元件與 JSX', dir: '2-1-components' },
			{ label: '2.2 狀態與 Hooks', dir: '2-2-state-hooks' },
			{ label: '2.3 Effect 與資料取得', dir: '2-3-effects' },
			{ label: '2.4 表單', dir: '2-4-forms' },
			{ label: '2.5 路由', dir: '2-5-routing' },
			{ label: '2.6 UI 與樣式', dir: '2-6-ui-styling' },
		],
	},
	{
		label: 'L3 與後端整合',
		dir: 'l3',
		sections: [
			{ label: '3.1 呼叫 API', dir: '3-1-api-calls' },
			{ label: '3.2 認證與 Session', dir: '3-2-auth-session' },
			{ label: '3.3 伺服器狀態', dir: '3-3-server-state' },
			{ label: '3.4 檔案上傳與下載', dir: '3-4-files' },
			{ label: '3.5 資料密集的畫面', dir: '3-5-data-heavy-ui' },
			{ label: '3.6 並發控制與衝突', dir: '3-6-concurrency' },
		],
	},
	{
		label: 'L4 品質與交付',
		dir: 'l4',
		sections: [
			{ label: '4.1 前端測試', dir: '4-1-testing' },
			{ label: '4.2 前端安全', dir: '4-2-security' },
			{ label: '4.3 無障礙與在地化', dir: '4-3-a11y-i18n' },
			{ label: '4.4 前端效能', dir: '4-4-performance' },
			{ label: '4.5 建置與部署', dir: '4-5-build-deploy' },
			{ label: '4.6 小結專案', dir: '4-6-wrap-up' },
		],
	},
];

export default fieldSidebar('frontend', levels);
