// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightSidebarTopics from 'starlight-sidebar-topics';
import mermaid from 'astro-mermaid';

import ai from './src/sidebar/ai.ts';
import backend from './src/sidebar/backend.ts';
import cs from './src/sidebar/cs.ts';
import cyber from './src/sidebar/cyber.ts';
import devops from './src/sidebar/devops.ts';
import testing from './src/sidebar/testing.ts';
import tools from './src/sidebar/tools.ts';

// https://astro.build/config
export default defineConfig({
	integrations: [
		// Must come before starlight() so ```mermaid fences are handled first.
		mermaid(),
		starlight({
			title: '軟體工程手冊',
			// Root locale only: content lives at the root, but UI strings
			// (search, "On this page", prev/next) render in Traditional Chinese.
			defaultLocale: 'root',
			locales: { root: { label: '繁體中文', lang: 'zh-TW' } },
			// The field switcher lives in the header (src/components/Header.astro).
			// Our Sidebar override takes precedence over the plugin's, which would
			// otherwise repeat the switcher above the sidebar.
			components: {
				Header: './src/components/Header.astro',
				Sidebar: './src/components/Sidebar.astro',
			},
			// Each field gets its own sidebar.
			plugins: [
				starlightSidebarTopics([
					{ label: '計算機科學', link: '/cs/', icon: 'laptop', items: cs },
					{ label: '後端', link: '/backend/', icon: 'seti:go', items: backend },
					{ label: '維運', link: '/devops/', icon: 'rocket', items: devops },
					{ label: '資安', link: '/cyber/', icon: 'warning', items: cyber },
					{ label: '測試', link: '/testing/', icon: 'approve-check', items: testing },
					{ label: 'AI', link: '/ai/', icon: 'puzzle', items: ai },
					{ label: '工具', link: '/tools/', icon: 'setting', items: tools },
				]),
			],
		}),
	],
});
