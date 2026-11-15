// @ts-check
/**
 * Expressive Code plugin: opt-in "Carbon" window for a single code block.
 *
 *   ```go carbon
 *   ```go title="main.go" carbon
 *
 * The block is rendered with the terminal frame (title bar with three dots and
 * a centred title) and its <figure> gets the `is-carbon` class. The window look
 * itself lives in src/styles/custom.css (section "Code blocks: Carbon window").
 */

/** Code blocks that asked for the Carbon window. */
const carbonBlocks = new WeakSet();

export function carbonWindow() {
	return {
		name: 'carbon-window',
		hooks: {
			/** @param {{ codeBlock: any }} context */
			preprocessMetadata: ({ codeBlock }) => {
				if (!codeBlock.metaOptions.getBoolean('carbon')) return;
				carbonBlocks.add(codeBlock);
				// Runs after the frames plugin has read `frame=...`, so this wins.
				codeBlock.props.frame = 'terminal';
			},
			/** @param {{ codeBlock: any, renderData: any }} context */
			postprocessRenderedBlock: ({ codeBlock, renderData }) => {
				if (!carbonBlocks.has(codeBlock)) return;
				const props = (renderData.blockAst.properties ??= {});
				const classes = props.className ?? [];
				props.className = [...(Array.isArray(classes) ? classes : [classes]), 'is-carbon'];
			},
		},
	};
}
