// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightSidebarTopics from 'starlight-sidebar-topics';
import mermaid from 'astro-mermaid';

import { carbonWindow } from './src/lib/ec-carbon.mjs';

import ai from './src/sidebar/ai.ts';
import backend from './src/sidebar/backend.ts';
import cs from './src/sidebar/cs.ts';
import cyber from './src/sidebar/cyber.ts';
import devops from './src/sidebar/devops.ts';
import testing from './src/sidebar/testing.ts';
import tools from './src/sidebar/tools.ts';
import frontend from './src/sidebar/frontend.ts';

// https://astro.build/config
export default defineConfig({
	site: 'https://site.knowledge-forest.online',
	integrations: [
		// Must come before starlight() so ```mermaid fences are handled first.
		mermaid(),
		starlight({
			title: '軟體工程手冊',
			logo: { src: './src/assets/logo.png', alt: '' },
			favicon: '/favicon.png',
			head: [{ tag: 'link', attrs: { rel: 'apple-touch-icon', href: '/favicon.png' } }],
			// Root locale only: content lives at the root, but UI strings
			// (search, "On this page", prev/next) render in Traditional Chinese.
			defaultLocale: 'root',
			locales: { root: { label: '繁體中文', lang: 'zh-TW' } },
			// Theme tokens and zh-Hant typography (see the file header).
			customCss: ['./src/styles/custom.css'],
			// Flat code blocks: thin border, no shadow, no terminal title bar.
			// Colours come from our --sl-* tokens (src/styles/custom.css), so the
			// blocks follow the site's light/dark gray scale.
			expressiveCode: {
				themes: ['github-dark', 'github-light'],
				// Untitled shell blocks render like any other block instead of a
				// terminal window. A `title="..."` still shows as an editor tab.
				defaultProps: { frame: 'code' },
				styleOverrides: {
					borderRadius: 'var(--hb-radius-s)',
					borderColor: 'var(--sl-color-gray-5)',
					codeBackground: ({ theme }) =>
						theme.type === 'dark' ? 'var(--sl-color-gray-6)' : 'var(--sl-color-gray-7)',
					codePaddingBlock: '1rem',
					codePaddingInline: '1.25rem',
					frames: {
						frameBoxShadowCssValue: 'none',
						// Title tab sits flush on the block: same background, no indicator.
						editorTabBarBackground: ({ resolveSetting }) => resolveSetting('codeBackground'),
						editorActiveTabBackground: ({ resolveSetting }) => resolveSetting('codeBackground'),
						editorActiveTabIndicatorTopColor: 'transparent',
						// The active tab is painted over the tab bar's bottom line, so it
						// draws that line itself; otherwise the line stops at the tab.
						editorActiveTabIndicatorBottomColor: 'var(--sl-color-gray-5)',
						editorTabBarBorderBottomColor: 'var(--sl-color-gray-5)',
					},
				},
				// `carbon` in a fence's meta opts that block into a window frame.
				plugins: [carbonWindow()],
				// Inline EC styles per page. The external ec.*.css build emits only one
				// of the hashed files, so pages link a missing one and render unstyled.
				emitExternalStylesheet: false,
			},
			// The field switcher lives in the header (src/components/Header.astro).
			// Our Sidebar override takes precedence over the plugin's, which would
			// otherwise repeat the switcher above the sidebar.
			components: {
				Header: './src/components/Header.astro',
				PageTitle: './src/components/PageTitle.astro',
				Sidebar: './src/components/Sidebar.astro',
				TableOfContents: './src/components/TableOfContents.astro',
			},
			// Each field gets its own sidebar.
			plugins: [
				starlightSidebarTopics([
					{ label: '計算機科學', link: '/cs/', icon: 'laptop', items: cs },
					{ label: '後端', link: '/backend/', icon: 'seti:go', items: backend },
					{ label: '前端', link: '/frontend/', icon: 'pencil', items: frontend },
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
