import { fieldSidebar, type Level } from './helpers';

// Mirrors docs/contents/cs.md. CS is grouped by topic instead of L0–L4, so each
// topic group takes the place of a level. The `dir` values are the canonical
// directory names under src/content/docs/cs/; articles must use them.
// Labels drop the outline's letter codes ("N", "N.4"); the codes stay in
// cs.md and in the `dir` values.
export const levels: Level[] = [
	{
		label: '計算機組成與程式執行',
		dir: 'c-computer',
		sections: [
			{ label: '程式怎麼跑', dir: 'c-1-program-execution' },
			{ label: '資料在機器裡的樣子', dir: 'c-2-data-representation' },
			{ label: 'CPU 與記憶體階層', dir: 'c-3-cpu-memory' },
		],
	},
	{
		label: '資料結構與演算法',
		dir: 'd-dsa',
		sections: [
			{ label: '複雜度分析', dir: 'd-1-complexity' },
			{ label: '基礎資料結構', dir: 'd-2-data-structures' },
			{ label: '基礎演算法', dir: 'd-3-algorithms' },
			{ label: '把它連到後端', dir: 'd-4-to-backend' },
		],
	},
	{
		label: '作業系統',
		dir: 'o-os',
		sections: [
			{ label: 'process 與 thread', dir: 'o-1-process-thread' },
			{ label: '記憶體管理', dir: 'o-2-memory' },
			{ label: '排程', dir: 'o-3-scheduling' },
			{ label: '檔案與 I/O', dir: 'o-4-file-io' },
			{ label: '並行基礎', dir: 'o-5-concurrency' },
			{ label: '隔離與容器的底層', dir: 'o-6-isolation' },
		],
	},
	{
		label: '計算機網路',
		dir: 'n-network',
		sections: [
			{ label: '分層模型', dir: 'n-1-layers' },
			{ label: '連結層與網路層', dir: 'n-2-link-network' },
			{ label: '傳輸層', dir: 'n-3-transport' },
			{ label: '應用層', dir: 'n-4-application' },
			{ label: '動手看封包', dir: 'n-5-packets' },
		],
	},
	{
		label: '命令列與開發工具',
		dir: 't-tools',
		sections: [
			{ label: 'Linux 命令列', dir: 't-1-linux-cli' },
			{ label: 'Shell 指令碼', dir: 't-2-shell' },
			{ label: '版本控制 Git', dir: 't-3-git' },
			{ label: '除錯與建置', dir: 't-4-debug-build' },
		],
	},
	{
		label: '綜合題',
		dir: 'z-capstone',
		sections: [{ label: '把它串起來', dir: 'z-1-putting-it-together' }],
	},
];

export default fieldSidebar('cs', levels);
