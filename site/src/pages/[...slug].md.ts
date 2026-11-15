// Serves each docs page as raw Markdown at /<slug>.md, for the "copy page"
// button in PageTitle.astro and for pasting a page into an LLM.
import type { APIRoute, GetStaticPaths } from 'astro';
import { getCollection, type CollectionEntry } from 'astro:content';

export const getStaticPaths = (async () => {
	const docs = await getCollection('docs');
	return docs.map((entry) => ({
		params: { slug: entry.id === '' ? 'index' : entry.id },
		props: { entry },
	}));
}) satisfies GetStaticPaths;

export const GET: APIRoute<{ entry: CollectionEntry<'docs'> }> = ({ props }) => {
	const { title, description } = props.entry.data;
	const parts = [`# ${title}`];
	if (description) parts.push(`> ${description}`);
	parts.push((props.entry.body ?? '').trim());
	return new Response(parts.join('\n\n') + '\n', {
		headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
	});
};
