import { fieldSidebar, type Level } from './helpers';

// Mirrors docs/contents/cyber.md. Cyber is grouped by engagement phase (P0–P7)
// instead of L0–L4, so each phase takes the place of a level. The `dir` values
// are the canonical directory names under src/content/docs/cyber/; articles
// must use them.
export const levels: Level[] = [
	{
		label: 'P0 準備',
		dir: 'p0-prep',
		sections: [
			{ label: '0.1 倫理、法律與授權', dir: '0-1-ethics-law' },
			{ label: '0.2 搭建自己的實驗室', dir: '0-2-lab' },
			{ label: '0.3 應用密碼學', dir: '0-3-crypto' },
		],
	},
	{
		label: 'P1 偵察與列舉',
		dir: 'p1-recon',
		sections: [
			{ label: '1.1 被動偵察（OSINT）', dir: '1-1-osint' },
			{ label: '1.2 主動掃描', dir: '1-2-scanning' },
			{ label: '1.3 服務與 Web 列舉', dir: '1-3-enumeration' },
			{ label: '1.4 防守視角', dir: '1-4-defense' },
		],
	},
	{
		label: 'P2 Web 滲透',
		dir: 'p2-web',
		sections: [
			{ label: '2.1 攔截與觀察', dir: '2-1-intercept' },
			{ label: '2.2 注入類', dir: '2-2-injection' },
			{ label: '2.3 跨站與請求偽造', dir: '2-3-cross-site' },
			{ label: '2.4 存取控制與認證', dir: '2-4-access-control' },
			{ label: '2.5 API 與現代前端', dir: '2-5-api-frontend' },
			{ label: '2.6 防守視角', dir: '2-6-defense' },
		],
	},
	{
		label: 'P3 系統與網路滲透',
		dir: 'p3-system',
		sections: [
			{ label: '3.1 利用與初始存取', dir: '3-1-initial-access' },
			{ label: '3.2 權限提升', dir: '3-2-privesc' },
			{ label: '3.3 橫向移動與內網', dir: '3-3-lateral' },
			{ label: '3.4 防守視角', dir: '3-4-defense' },
		],
	},
	{
		label: 'P4 進階技術',
		dir: 'p4-advanced',
		sections: [
			{ label: '4.1 逆向工程入門', dir: '4-1-reversing' },
			{ label: '4.2 二進位漏洞概念', dir: '4-2-binary' },
			{ label: '4.3 雲與容器攻擊面', dir: '4-3-cloud-container' },
			{ label: '4.4 CTF 方法', dir: '4-4-ctf' },
		],
	},
	{
		label: 'P5 紅隊行動',
		dir: 'p5-red-team',
		sections: [
			{ label: '5.1 對手思維與 ATT&CK', dir: '5-1-attack' },
			{ label: '5.2 行動規劃', dir: '5-2-planning' },
			{ label: '5.3 紫隊協作', dir: '5-3-purple' },
		],
	},
	{
		label: 'P6 防禦與應變',
		dir: 'p6-defense',
		sections: [
			{ label: '6.1 日誌與偵測', dir: '6-1-detection' },
			{ label: '6.2 強化', dir: '6-2-hardening' },
			{ label: '6.3 事故應變與鑑識', dir: '6-3-incident' },
		],
	},
	{
		label: 'P7 報告與職涯',
		dir: 'p7-report',
		sections: [
			{ label: '7.1 寫滲透測試報告', dir: '7-1-report' },
			{ label: '7.2 漏洞賞金與揭露', dir: '7-2-disclosure' },
			{ label: '7.3 學習路線與證照', dir: '7-3-career' },
		],
	},
];

export default fieldSidebar('cyber', levels);
