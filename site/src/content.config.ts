import { defineCollection, z } from 'astro:content';
import { docsLoader } from '@astrojs/starlight/loaders';
import { docsSchema } from '@astrojs/starlight/schema';

export const collections = {
	docs: defineCollection({
		loader: docsLoader(),
		schema: docsSchema({
			// Tools articles go stale as products update; `verified` records which
			// tool version an article was last checked against.
			extend: z.object({
				verified: z
					.object({
						tool: z.string(),
						version: z.string(),
						date: z.coerce.date(),
					})
					.optional(),
			}),
		}),
	}),
};
