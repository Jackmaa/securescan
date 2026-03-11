<script lang="ts">
	import { page } from '$app/stores';
	import { getFixes, acceptFix, rejectFix } from '$lib/api';
	import type { Fix } from '$lib/types';

	/**
	 * Fix review page: lets the user accept/reject remediation suggestions.
	 *
	 * Why fixes are reviewed separately from findings:
	 * - Fix payloads can include multi-line code snippets and explanations.
	 * - The workflow is decision-based (accept/reject), so separating keeps the dashboard focused.
	 */
	let scanId = $derived($page.params.id);
	let fixes = $state<Fix[]>([]);
	let loading = $state(true);

	/** Loads the current list of fixes from the backend for this scan. */
	async function loadFixes() {
		loading = true;
		fixes = await getFixes(scanId);
		loading = false;
	}

	$effect(() => {
		// Initial load when the route mounts.
		loadFixes();
	});

	/**
	 * handleAccept updates backend status then reflects it in local UI state.
	 *
	 * Note: after mutating a nested object (`fix.status`), we reassign `fixes` to a new array
	 * to ensure reactive updates propagate.
	 */
	async function handleAccept(fix: Fix) {
		await acceptFix(fix.id);
		fix.status = 'accepted';
		fixes = [...fixes];
	}

	/** Same as accept, but transitions status to rejected. */
	async function handleReject(fix: Fix) {
		await rejectFix(fix.id);
		fix.status = 'rejected';
		fixes = [...fixes];
	}
</script>

<svelte:head>
	<title>Fix Review | SecureScan</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">Review Fixes</h1>
		<a
			href="/scan/{scanId}/dashboard"
			class="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
		>
			Back to Dashboard
		</a>
	</div>

	{#if loading}
		<div class="py-16 text-center text-[var(--color-text-muted)]">Loading fixes...</div>
	{:else if fixes.length === 0}
		<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] px-6 py-16 text-center text-[var(--color-text-muted)]">
			No fixes generated for this scan.
		</div>
	{:else}
		{#each fixes as fix}
			<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6">
				<div class="mb-3 flex items-center justify-between">
					<div>
						<span class="rounded bg-[var(--color-primary)]/20 px-2 py-0.5 text-xs font-medium text-[var(--color-primary)]">
							{fix.fix_type}
						</span>
						<span class="ml-2 text-sm text-[var(--color-text)]">{fix.description}</span>
					</div>
					<span class="text-xs capitalize text-[var(--color-text-muted)]">{fix.status}</span>
				</div>

				<div class="mb-3 font-mono text-xs text-[var(--color-text-muted)]">
					{fix.file_path}{fix.line_start ? `:${fix.line_start}` : ''}
				</div>

				{#if fix.original_code || fix.fixed_code}
					<div class="grid grid-cols-2 gap-2 text-xs">
						{#if fix.original_code}
							<div>
								<div class="mb-1 text-[var(--color-critical)]">Original</div>
								<pre class="overflow-x-auto rounded-lg bg-[var(--color-bg)] p-3">{fix.original_code}</pre>
							</div>
						{/if}
						{#if fix.fixed_code}
							<div>
								<div class="mb-1 text-[var(--color-success)]">Fixed</div>
								<pre class="overflow-x-auto rounded-lg bg-[var(--color-bg)] p-3">{fix.fixed_code}</pre>
							</div>
						{/if}
					</div>
				{/if}

				{#if fix.explanation}
					<div class="mt-3 text-sm text-[var(--color-text-muted)]">{fix.explanation}</div>
				{/if}

				{#if fix.status === 'pending'}
					<div class="mt-4 flex gap-2">
						<button
							onclick={() => handleAccept(fix)}
							class="rounded-lg bg-[var(--color-success)] px-4 py-1.5 text-sm font-medium text-white transition-opacity hover:opacity-90"
						>
							Accept
						</button>
						<button
							onclick={() => handleReject(fix)}
							class="rounded-lg border border-[var(--color-border)] px-4 py-1.5 text-sm transition-colors hover:bg-[var(--color-surface-hover)]"
						>
							Reject
						</button>
					</div>
				{/if}
			</div>
		{/each}
	{/if}
</div>
