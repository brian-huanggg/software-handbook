import { fieldSidebar, type Level } from './helpers';

// Mirrors docs/contents/backend.md. The `dir` values are the canonical
// directory names under src/content/docs/backend/; articles must use them.
// Backend starts at L1: the foundations it assumes live in the cs field (cs.ts).
export const levels: Level[] = [
	{
		label: 'L1 入門',
		dir: 'l1',
		sections: [
			{ label: '1.1 Go 語言核心', dir: '1-1-go-core' },
			{ label: '1.2 HTTP 服務與 API', dir: '1-2-http-api' },
			{ label: '1.3 關聯式資料庫入門', dir: '1-3-relational-db' },
			{ label: '1.4 認證與基礎安全', dir: '1-4-auth-basics' },
			{ label: '1.5 測試入門與程式碼品質', dir: '1-5-testing' },
			{ label: '1.6 小結專案', dir: '1-6-project' },
		],
	},
	{
		label: 'L2 中階',
		dir: 'l2',
		sections: [
			{ label: '2.1 Go 並行程式設計', dir: '2-1-concurrency' },
			{ label: '2.2 資料庫深入', dir: '2-2-database' },
			{ label: '2.3 快取', dir: '2-3-cache' },
			{ label: '2.4 非同步與訊息佇列', dir: '2-4-messaging' },
			{ label: '2.5 API 設計進階', dir: '2-5-api-design' },
			{ label: '2.6 認證授權進階', dir: '2-6-auth-advanced' },
			{ label: '2.7 容器與交付', dir: '2-7-containers' },
			{ label: '2.8 可觀測性基礎', dir: '2-8-observability' },
			{ label: '2.9 小結專案', dir: '2-9-project' },
		],
	},
	{
		label: 'L3 資深',
		dir: 'l3',
		sections: [
			{ label: '3.1 分散式系統基礎', dir: '3-1-distributed-basics' },
			{ label: '3.2 跨服務資料一致性', dir: '3-2-data-consistency' },
			{ label: '3.3 服務架構', dir: '3-3-service-architecture' },
			{ label: '3.4 流量入口：反向代理與負載平衡', dir: '3-4-traffic-entry' },
			{ label: '3.5 韌性設計', dir: '3-5-resilience' },
			{ label: '3.6 效能調校', dir: '3-6-performance' },
			{ label: '3.7 資料底層原理', dir: '3-7-data-internals' },
			{ label: '3.8 雲原生', dir: '3-8-cloud-native' },
			{ label: '3.9 系統設計實戰', dir: '3-9-system-design' },
		],
	},
	{
		label: 'L4 應用系統',
		dir: 'l4',
		sections: [
			{ label: '4.1 檔案上傳與儲存', dir: '4-1-file-upload' },
			{ label: '4.2 權限系統：RBAC 到細粒度授權', dir: '4-2-authorization' },
			{ label: '4.3 審計日誌', dir: '4-3-audit-log' },
			{ label: '4.4 通知系統', dir: '4-4-notification' },
			{ label: '4.5 支付與訂閱計費', dir: '4-5-billing' },
			{ label: '4.6 串流與即時通訊', dir: '4-6-streaming' },
			{ label: '4.7 搜尋', dir: '4-7-search' },
			{ label: '4.8 其他常見的 SaaS 功能', dir: '4-8-saas-misc' },
			{ label: '4.9 小結專案', dir: '4-9-project' },
		],
	},
	{
		label: 'L5 架構',
		dir: 'l5',
		sections: [
			{ label: '5.1 可靠性工程', dir: '5-1-reliability' },
			{ label: '5.2 安全與合規', dir: '5-2-security-compliance' },
			{ label: '5.3 資料與平台', dir: '5-3-data-platform' },
			{ label: '5.4 成本與商業思維', dir: '5-4-cost-business' },
			{ label: '5.5 技術決策與團隊', dir: '5-5-tech-decisions' },
			{ label: '5.6 終點專題：真實世界案例讀法', dir: '5-6-case-studies' },
		],
	},
];

export default fieldSidebar('backend', levels);
