import { fieldSidebar, type Level } from './helpers';

// The AI field is one topic with three tracks. Each track mirrors its own outline
// in docs/contents/ai/ (foundations.md, engineering.md, ai-agents.md; the agents
// outline is prefixed so tools do not mistake it for an agent-instructions file),
// and its articles live under
// src/content/docs/ai/<track>/<level>/<section>/. The `dir` values below are the
// canonical directory names; articles must use them.

interface Track {
	label: string;
	dir: string;
	levels: Level[];
}

const foundations: Level[] = [
	{
		label: 'L1 數學與工具',
		dir: 'l1',
		sections: [
			{ label: '1.1 Python 科學計算環境', dir: '1-1-python-scientific' },
			{ label: '1.2 線性代數', dir: '1-2-linear-algebra' },
			{ label: '1.3 微積分與最佳化', dir: '1-3-calculus-optimization' },
			{ label: '1.4 機率與統計', dir: '1-4-probability-statistics' },
			{ label: '1.5 資料處理', dir: '1-5-data-processing' },
		],
	},
	{
		label: 'L2 經典機器學習',
		dir: 'l2',
		sections: [
			{ label: '2.1 機器學習的框架', dir: '2-1-ml-framework' },
			{ label: '2.2 迴歸', dir: '2-2-regression' },
			{ label: '2.3 分類', dir: '2-3-classification' },
			{ label: '2.4 樹模型與集成學習', dir: '2-4-trees-ensembles' },
			{ label: '2.5 非監督式學習', dir: '2-5-unsupervised' },
			{ label: '2.6 模型評估與選擇', dir: '2-6-model-evaluation' },
			{ label: '2.7 特徵工程與機器學習流程', dir: '2-7-feature-engineering' },
		],
	},
	{
		label: 'L3 深度學習',
		dir: 'l3',
		sections: [
			{ label: '3.1 神經網路基礎', dir: '3-1-neural-networks' },
			{ label: '3.2 PyTorch', dir: '3-2-pytorch' },
			{ label: '3.3 訓練技巧', dir: '3-3-training-techniques' },
			{ label: '3.4 卷積神經網路', dir: '3-4-cnn' },
			{ label: '3.5 序列模型', dir: '3-5-sequence-models' },
			{ label: '3.6 表示學習與注意力', dir: '3-6-representation-attention' },
			{ label: '3.7 生成模型', dir: '3-7-generative-models' },
		],
	},
	{
		label: 'L4 大型語言模型',
		dir: 'l4',
		sections: [
			{ label: '4.1 Transformer 架構', dir: '4-1-transformer' },
			{ label: '4.2 Tokenization', dir: '4-2-tokenization' },
			{ label: '4.3 預訓練', dir: '4-3-pretraining' },
			{ label: '4.4 微調與對齊', dir: '4-4-finetuning-alignment' },
			{ label: '4.5 推論與最佳化', dir: '4-5-inference-optimization' },
			{ label: '4.6 多模態與前沿架構', dir: '4-6-multimodal-frontier' },
			{ label: '4.7 評估、可解釋性與安全', dir: '4-7-eval-interpretability-safety' },
		],
	},
];

const engineering: Level[] = [
	{
		label: 'L1 呼叫模型',
		dir: 'l1',
		sections: [
			{ label: '1.1 LLM 是什麼（給工程師的版本）', dir: '1-1-llm-for-engineers' },
			{ label: '1.2 呼叫 LLM API', dir: '1-2-llm-api' },
			{ label: '1.3 提示工程', dir: '1-3-prompt-engineering' },
			{ label: '1.4 結構化輸出', dir: '1-4-structured-output' },
			{ label: '1.5 token 與成本', dir: '1-5-tokens-cost' },
		],
	},
	{
		label: 'L2 檢索增強',
		dir: 'l2',
		sections: [
			{ label: '2.1 嵌入與語意搜尋', dir: '2-1-embeddings-search' },
			{ label: '2.2 向量資料庫', dir: '2-2-vector-db' },
			{ label: '2.3 文件處理', dir: '2-3-document-processing' },
			{ label: '2.4 RAG 系統', dir: '2-4-rag' },
			{ label: '2.5 進階檢索', dir: '2-5-advanced-retrieval' },
		],
	},
	{
		label: 'L3 品質與上線',
		dir: 'l3',
		sections: [
			{ label: '3.1 評估', dir: '3-1-evaluation' },
			{ label: '3.2 評估驅動開發', dir: '3-2-eval-driven-dev' },
			{ label: '3.3 防護與安全', dir: '3-3-guardrails-security' },
			{ label: '3.4 效能與可靠性', dir: '3-4-performance-reliability' },
			{ label: '3.5 可觀測性', dir: '3-5-observability' },
		],
	},
	{
		label: 'L4 自架與架構',
		dir: 'l4',
		sections: [
			{ label: '4.1 自架開源模型', dir: '4-1-self-hosting' },
			{ label: '4.2 GPU 與算力規劃', dir: '4-2-gpu-planning' },
			{ label: '4.3 微調的時機', dir: '4-3-when-to-finetune' },
			{ label: '4.4 LLMOps', dir: '4-4-llmops' },
			{ label: '4.5 架構與治理', dir: '4-5-architecture-governance' },
		],
	},
];

const agents: Level[] = [
	{
		label: 'L1 工具與迴圈',
		dir: 'l1',
		sections: [
			{ label: '1.1 什麼是代理人', dir: '1-1-what-is-agent' },
			{ label: '1.2 工具呼叫', dir: '1-2-tool-calling' },
			{ label: '1.3 代理人迴圈', dir: '1-3-agent-loop' },
			{ label: '1.4 工具設計', dir: '1-4-tool-design' },
			{ label: '1.5 Model Context Protocol（MCP）', dir: '1-5-mcp' },
		],
	},
	{
		label: 'L2 記憶與規劃',
		dir: 'l2',
		sections: [
			{ label: '2.1 上下文管理', dir: '2-1-context-management' },
			{ label: '2.2 記憶', dir: '2-2-memory' },
			{ label: '2.3 規劃與推理', dir: '2-3-planning-reasoning' },
			{ label: '2.4 長時間任務', dir: '2-4-long-running-tasks' },
		],
	},
	{
		label: 'L3 多代理人與工作流程',
		dir: 'l3',
		sections: [
			{ label: '3.1 工作流程模式', dir: '3-1-workflow-patterns' },
			{ label: '3.2 多代理人系統', dir: '3-2-multi-agent' },
			{ label: '3.3 人機協作', dir: '3-3-human-in-the-loop' },
			{ label: '3.4 程式碼代理人', dir: '3-4-coding-agents' },
			{ label: '3.5 框架與 SDK', dir: '3-5-frameworks-sdks' },
		],
	},
	{
		label: 'L4 正式環境的代理人',
		dir: 'l4',
		sections: [
			{ label: '4.1 代理人評估', dir: '4-1-agent-evaluation' },
			{ label: '4.2 代理人安全', dir: '4-2-agent-security' },
			{ label: '4.3 沙箱與權限', dir: '4-3-sandbox-permissions' },
			{ label: '4.4 營運與治理', dir: '4-4-operations-governance' },
		],
	},
];

export const tracks: Track[] = [
	{ label: '理論基礎', dir: 'foundations', levels: foundations },
	{ label: '應用工程', dir: 'engineering', levels: engineering },
	{ label: '代理人', dir: 'agents', levels: agents },
];

// Tracks become the top-level sidebar groups; a track with no articles yet is
// hidden, the same way fieldSidebar hides empty levels and sections.
export default tracks
	.map((track) => ({
		label: track.label,
		items: fieldSidebar(`ai/${track.dir}`, track.levels),
	}))
	.filter((track) => track.items.length > 0);
