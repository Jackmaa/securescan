import { sveltekit } from '@sveltejs/kit/vite'; // SvelteKit integration for Vite (routing, SSR, etc.).
import tailwindcss from '@tailwindcss/vite'; // Tailwind v4 Vite plugin (build-time CSS processing).
import { defineConfig } from 'vite'; // Typed Vite config helper.

/**
 * Vite configuration for the SecureScan frontend.
 *
 * Key choices:
 * - Tailwind is registered before SvelteKit so CSS processing is available to Svelte compilation.
 * - A dev proxy forwards `/api/*` to the Go backend, allowing the frontend to call
 *   relative API paths without CORS complexity during local development.
 */
export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	server: {
		proxy: {
			'/api': 'http://localhost:3000'
		}
	}
});
