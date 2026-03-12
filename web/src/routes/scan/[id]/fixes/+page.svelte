<script lang="ts">
	import { page } from '$app/stores';
	import { getFixes, acceptFix, rejectFix, applyFixes } from '$lib/api';
	import type { Fix } from '$lib/types';

	let scanId = $derived($page.params.id);
	let fixes = $state<Fix[]>([]);
	let loading = $state(true);
	let applying = $state(false);
	let applyResult = $state('');

	async function loadFixes() {
		loading = true;
		fixes = await getFixes(scanId);
		loading = false;
	}

	$effect(() => {
		loadFixes();
	});

	async function handleAccept(fix: Fix) {
		await acceptFix(fix.id);
		fix.status = 'accepted';
		fixes = [...fixes];
	}

	async function handleReject(fix: Fix) {
		await rejectFix(fix.id);
		fix.status = 'rejected';
		fixes = [...fixes];
	}

	let hasAccepted = $derived(fixes.some(f => f.status === 'accepted'));

	async function handleApplyFixes() {
		applying = true;
		applyResult = '';
		try {
			const result = await applyFixes(scanId);
			applyResult = result.message;
			await loadFixes();
		} catch (e) {
			applyResult = e instanceof Error ? e.message : 'Failed to apply fixes';
		} finally {
			applying = false;
		}
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

		{#if hasAccepted}
			<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 text-center">
				<button
					onclick={handleApplyFixes}
					disabled={applying}
					class="rounded-lg bg-[var(--color-primary)] px-6 py-3 font-medium text-white transition-colors hover:bg-[var(--color-primary-hover)] disabled:opacity-50"
				>
					{applying ? 'Applying...' : 'Apply Accepted Fixes (Create Branch & PR)'}
				</button>
				{#if applyResult}
					<div class="mt-3 text-sm text-[var(--color-text-muted)]">{applyResult}</div>
				{/if}
			</div>
		{/if}
	{/if}
</div>
