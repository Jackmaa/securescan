<script lang="ts">
	import { page } from '$app/stores';
	import { generateReport as apiGenerateReport } from '$lib/api';

	let scanId = $derived($page.params.id);
	let generating = $state(false);
	let downloadUrl = $state('');
	let error = $state('');

	async function generateReport() {
		generating = true;
		error = '';
		try {
			await apiGenerateReport(scanId);
			downloadUrl = `/api/scans/${scanId}/report/download`;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to generate report';
		} finally {
			generating = false;
		}
	}
</script>

<svelte:head>
	<title>Report | SecureScan</title>
</svelte:head>

<div class="mx-auto max-w-2xl space-y-6 py-16">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">Security Report</h1>
		<a
			href="/scan/{scanId}/dashboard"
			class="text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text)]"
		>
			Back to Dashboard
		</a>
	</div>

	<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-8 text-center">
		<div class="mb-4 text-4xl">&#128196;</div>
		<p class="mb-6 text-[var(--color-text-muted)]">
			Generate a comprehensive security report with all findings, OWASP mapping, and fix recommendations.
		</p>

		<button
			onclick={generateReport}
			disabled={generating}
			class="rounded-lg bg-[var(--color-primary)] px-6 py-3 font-medium text-white transition-colors hover:bg-[var(--color-primary-hover)] disabled:opacity-50"
		>
			{generating ? 'Generating...' : 'Generate PDF Report'}
		</button>

		{#if downloadUrl}
			<div class="mt-4">
				<a
					href={downloadUrl}
					class="text-[var(--color-primary)] hover:underline"
					download
				>
					Download Report
				</a>
			</div>
		{/if}
	</div>
</div>
