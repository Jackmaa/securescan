<script lang="ts">
	/**
	 * RepoInput collects a Git URL and emits a submit event to the parent.
	 *
	 * Why it delegates submission to the parent:
	 * - The parent route owns API calls, navigation, and error handling.
	 * - This component stays purely UI + input validation, which improves reuse and testability.
	 */
	interface Props {
		onsubmit: (name: string, url: string) => void;
		loading: boolean;
	}

	let { onsubmit, loading }: Props = $props();

	let url = $state('');
	let name = $derived(extractRepoName(url));

	/**
	 * extractRepoName infers a reasonable default project name from the URL.
	 *
	 * Why we do this:
	 * - Most users paste `https://github.com/org/repo` and expect a good default name.
	 * - We avoid a second input field until it becomes necessary.
	 */
	function extractRepoName(repoUrl: string): string {
		try {
			const parts = repoUrl.replace(/\.git$/, '').split('/');
			return parts[parts.length - 1] || '';
		} catch {
			return '';
		}
	}

	/**
	 * handleSubmit prevents the browser form submission and calls the provided callback.
	 *
	 * Minimal validation lives here (non-empty URL + derived name) so we can keep the
	 * button/inputs responsive; deeper validation happens on the backend.
	 */
	function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!url.trim() || !name) return;
		onsubmit(name, url.trim());
	}
</script>

<form onsubmit={handleSubmit} class="space-y-4">
	<div>
		<label for="repo-url" class="mb-2 block text-sm font-medium text-[var(--color-text-muted)]">
			Git Repository URL
		</label>
		<input
			id="repo-url"
			type="url"
			bind:value={url}
			placeholder="https://github.com/owner/repo"
			required
			disabled={loading}
			class="w-full rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] px-4 py-3
				text-[var(--color-text)] placeholder-[var(--color-text-muted)]/50
				focus:border-[var(--color-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--color-primary)]
				disabled:opacity-50"
		/>
	</div>

	{#if name}
		<div class="text-sm text-[var(--color-text-muted)]">
			Project: <span class="font-medium text-[var(--color-text)]">{name}</span>
		</div>
	{/if}

	<button
		type="submit"
		disabled={loading || !url.trim()}
		class="w-full rounded-lg bg-[var(--color-primary)] px-4 py-3 font-medium text-white
			transition-colors hover:bg-[var(--color-primary-hover)]
			disabled:cursor-not-allowed disabled:opacity-50"
	>
		{#if loading}
			<span class="inline-flex items-center gap-2">
				<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
					<circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" class="opacity-25" />
					<path fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" class="opacity-75" />
				</svg>
				Cloning & Scanning...
			</span>
		{:else}
			Scan Repository
		{/if}
	</button>
</form>
