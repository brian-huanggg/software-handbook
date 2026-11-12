import { fieldSidebar, type Level } from './helpers';

// Mirrors docs/contents/devops.md. The `dir` values are the canonical
// directory names under src/content/docs/devops/; articles must use them.
// Only levels and sections that have articles are listed so far: add the rest
// here, in the same shape as backend.ts, as they get written.
export const levels: Level[] = [
	{
		label: 'L1 交付流水線',
		dir: 'l1',
		sections: [{ label: '1.2 容器映像', dir: '1-2-container-image' }],
	},
];

export default fieldSidebar('devops', levels);
