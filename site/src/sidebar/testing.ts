import { fieldSidebar, type Level } from './helpers';

// Mirrors docs/contents/testing.md. The `dir` values are the canonical
// directory names under src/content/docs/testing/; articles must use them.
export const levels: Level[] = [
	{
		label: 'L0 品質思維',
		dir: 'l0',
		sections: [
			{ label: '0.1 品質是什麼', dir: '0-1-what-is-quality' },
			{ label: '0.2 QA 的角色', dir: '0-2-qa-role' },
			{ label: '0.3 測試的基本原則', dir: '0-3-testing-principles' },
			{ label: '0.4 小結', dir: '0-4-wrap-up' },
		],
	},
	{
		label: 'L1 測試設計',
		dir: 'l1',
		sections: [
			{ label: '1.1 從需求開始', dir: '1-1-requirements' },
			{ label: '1.2 黑箱測試技術', dir: '1-2-black-box' },
			{ label: '1.3 白箱測試技術', dir: '1-3-white-box' },
			{ label: '1.4 經驗式與探索式測試', dir: '1-4-exploratory' },
			{ label: '1.5 缺陷管理', dir: '1-5-defect-management' },
			{ label: '1.6 小結', dir: '1-6-wrap-up' },
		],
	},
	{
		label: 'L2 自動化分層',
		dir: 'l2',
		sections: [
			{ label: '2.1 單元測試進階', dir: '2-1-unit-testing' },
			{ label: '2.2 整合測試', dir: '2-2-integration-testing' },
			{ label: '2.3 API 測試', dir: '2-3-api-testing' },
			{ label: '2.4 契約測試', dir: '2-4-contract-testing' },
			{ label: '2.5 UI 與 E2E 測試', dir: '2-5-ui-e2e' },
			{ label: '2.6 行為驅動開發（BDD）', dir: '2-6-bdd' },
			{ label: '2.7 小結', dir: '2-7-wrap-up' },
		],
	},
	{
		label: 'L3 測試套件工程',
		dir: 'l3',
		sections: [
			{ label: '3.1 測試資料管理', dir: '3-1-test-data' },
			{ label: '3.2 測試環境', dir: '3-2-test-environments' },
			{ label: '3.3 不穩定測試（flaky tests）', dir: '3-3-flaky-tests' },
			{ label: '3.4 衡量測試的有效性', dir: '3-4-test-effectiveness' },
			{ label: '3.5 測試速度與規模', dir: '3-5-speed-and-scale' },
			{ label: '3.6 接進 CI 與品質閘門', dir: '3-6-ci-quality-gates' },
			{ label: '3.7 小結', dir: '3-7-wrap-up' },
		],
	},
	{
		label: 'L4 非功能與品質策略',
		dir: 'l4',
		sections: [
			{ label: '4.1 效能測試', dir: '4-1-performance' },
			{ label: '4.2 進階自動化技術', dir: '4-2-advanced-automation' },
			{ label: '4.3 韌性與可靠性測試', dir: '4-3-resilience' },
			{ label: '4.4 安全與無障礙測試', dir: '4-4-security-a11y' },
			{ label: '4.5 在正式環境中測試', dir: '4-5-testing-in-production' },
			{ label: '4.6 測試策略與規劃', dir: '4-6-test-strategy' },
			{ label: '4.7 帶領品質工作', dir: '4-7-quality-leadership' },
			{ label: '4.8 收尾', dir: '4-8-capstone' },
		],
	},
];

export default fieldSidebar('testing', levels);
