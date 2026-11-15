/** URL of a docs entry's raw Markdown, served by src/pages/[...slug].md.ts. */
export function markdownPath(id: string): string {
	return `/${id === '' ? 'index' : id}.md`;
}
