<script lang="ts">
	import { goto } from '$app/navigation';
	import { createProject, triggerScan } from '$lib/api';
	import RepoInput from '$lib/components/RepoInput.svelte';

	/**
	 * Home page: accepts a repository URL and starts a scan.
	 *
	 * Import rationale:
	 * - `goto`: programmatic navigation after the scan is created.
	 * - `createProject/triggerScan`: backend calls are centralized in `$lib/api`.
	 * - `RepoInput`: isolates the form UI from orchestration and error handling.
	 */

	let loading = $state(false);
	let error = $state('');

	/**
	 * handleSubmit is called by RepoInput when the user submits a repo URL.
	 *
	 * Flow:
	 * - Create project (stages repo server-side).
	 * - Trigger scan (starts async pipeline).
	 * - Navigate to the scan progress page.
	 *
	 * Why we keep it sequential:
	 * - Triggering a scan requires a project ID.
	 * - If creation fails, we can show a precise error without starting a scan.
	 */
	async function handleSubmit(name: string, url: string) {
		loading = true;
		error = '';

		try {
			const project = await createProject(name, 'git', url);
			const scan = await triggerScan(project.id);
			goto(`/scan/${scan.id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Something went wrong';
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>SecureScan</title>
</svelte:head>

<div class="flex min-h-[70vh] flex-col items-center justify-center">
	<div class="mb-12 text-center">
		<h1 class="mb-4 text-5xl font-bold tracking-tight">
			<span class="text-[var(--color-primary)]">Secure</span>Scan
		</h1>
		<p class="max-w-lg text-lg text-[var(--color-text-muted)]">
			Submit a Git repository to scan for security vulnerabilities.
			Results are mapped to the OWASP Top 10:2025 with automated fix suggestions.
		</p>
	</div>

	<div class="w-full max-w-xl">
		<RepoInput onsubmit={handleSubmit} {loading} />

		{#if error}
			<div class="mt-4 rounded-lg border border-[var(--color-critical)]/30 bg-[var(--color-critical)]/10 px-4 py-3 text-sm text-[var(--color-critical)]">
				{error}
			</div>
		{/if}
	</div>

	<div class="mt-16 grid grid-cols-3 gap-8 text-center text-sm text-[var(--color-text-muted)]">
		<div>
			<div class="mb-2 text-2xl font-bold text-[var(--color-text)]">4</div>
			Security Tools
		</div>
		<div>
			<div class="mb-2 text-2xl font-bold text-[var(--color-text)]">10</div>
			OWASP Categories
		</div>
		<div>
			<div class="mb-2 text-2xl font-bold text-[var(--color-text)]">AI</div>
			Fix Suggestions
		</div>
	</div>
</div>
