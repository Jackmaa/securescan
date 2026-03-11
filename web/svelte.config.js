import adapter from '@sveltejs/adapter-auto'; // Chooses a deployment adapter automatically (dev/preview/platform-aware).
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte'; // Enables TS/Tailwind/Vite preprocessing in Svelte files.

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// adapter-auto is convenient while iterating; production deployments can switch to
		// a target-specific adapter (node, vercel, cloudflare, etc.) for tighter control.
		adapter: adapter()
	}
};

export default config;
