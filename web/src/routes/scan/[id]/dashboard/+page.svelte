$<script lang="ts">
	import { page } from '$app/stores';
	import { getScanStats, getFindings } from '$lib/api';
	import type { ScanStats, FindingResult } from '$lib/types';

	/**
	 * Scan dashboard page: summary metrics + a filtered findings list.
	 *
	 * Why this page fetches two endpoints:
	 * - `getScanStats` provides a fast, aggregated overview for charts/tiles.
	 * - `getFindings` provides the actual items for a table view, with server-side filters.
	 *
	 * The server performs grouping/filtering so the frontend doesn’t need to download
	 * the full findings dataset just to compute counts.
	 */
	let scanId = $derived($page.params.id);
	let stats = $state<ScanStats | null>(null);
	let findings = $state<FindingResult | null>(null);
	let loading = $state(true);
	let severityFilter = $state('');
	let owaspFilter = $state('');
	let toolFilter = $state('');

	// Human-friendly OWASP names for display in the category list.
	const owaspLabels: Record<string, string> = {
		A01: 'Broken Access Control',
		A02: 'Cryptographic Failures',
		A03: 'Injection',
		A04: 'Insecure Design',
		A05: 'Security Misconfiguration',
		A06: 'Vulnerable Components',
		A07: 'Auth Failures',
		A08: 'Data Integrity Failures',
		A09: 'Logging Failures',
		A10: 'SSRF'
	};

	// Central place for severity → color mapping so badges/bars stay consistent.
	const severityColors: Record<string, string> = {
		critical: 'var(--color-critical)',
		high: 'var(--color-high)',
		medium: 'var(--color-medium)',
		low: 'var(--color-low)',
		info: 'var(--color-info)'
	};

	/**
	 * loadData fetches stats + findings for the current filter state.
	 *
	 * We build `params` as a plain record so it can be fed directly into URLSearchParams
	 * in the API client.
	 */
	async function loadData() {
		loading = true;
		const params: Record<string, string> = {};
		if (severityFilter) params.severity = severityFilter;
		if (owaspFilter) params.owasp = owaspFilter;
		if (toolFilter) params.tool = toolFilter;

		[stats, findings] = await Promise.all([
			getScanStats(scanId),
			getFindings(scanId, params)
		]);
		loading = false;
	}

	$effect(() => {
		// Initial load when the route mounts.
		loadData();
	});

	// Reload when filters change
	$effect(() => {
		// Referencing these values establishes reactive dependencies in Svelte 5 runes.
		void severityFilter;
		void owaspFilter;
		void toolFilter;
		loadData();
	});

	/**
	 * gradeColor maps A/B/C/D to theme colors.
	 *
	 * This is presentation-focused (not a security model); the score/grade computation
	 * lives server-side so it can evolve without redeploying the frontend.
	 */
	function gradeColor(grade: string | undefined): string {
		switch (grade) {
			case 'A': return 'var(--color-success)';
			case 'B': return 'var(--color-low)';
			case 'C': return 'var(--color-medium)';
			case 'D': return 'var(--color-high)';
			default: return 'var(--color-critical)';
		}
	}
</script>

<svelte:head>
	<title>Dashboard | SecureScan</title>
</svelte:head>

{#if loading && !stats}
	<div class="flex items-center justify-center py-16">
		<div class="text-[var(--color-text-muted)]">Loading dashboard...</div>
	</div>
{:else if stats}
	<div class="space-y-6">
		<!-- Header -->
		<div class="flex items-center justify-between">
			<h1 class="text-2xl font-bold">Scan Results</h1>
			<div class="flex gap-3">
				<a
					href="/scan/{scanId}/fixes"
					class="rounded-lg border border-[var(--color-border)] px-4 py-2 text-sm transition-colors hover:bg-[var(--color-surface-hover)]"
				>
					Review Fixes
				</a>
				<a
					href="/scan/{scanId}/report"
					class="rounded-lg bg-[var(--color-primary)] px-4 py-2 text-sm text-white transition-colors hover:bg-[var(--color-primary-hover)]"
				>
					Generate Report
				</a>
			</div>
		</div>

		<!-- Score + Severity row -->
		<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
			<!-- Score Gauge -->
			<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6 text-center">
				<div class="text-sm text-[var(--color-text-muted)]">Security Score</div>
				<div class="mt-2 text-6xl font-bold" style="color: {gradeColor(stats.grade)}">
					{stats.grade || '?'}
				</div>
				<div class="mt-1 text-3xl font-light text-[var(--color-text)]">
					{stats.score ?? '—'}<span class="text-lg text-[var(--color-text-muted)]">/100</span>
				</div>
				<div class="mt-2 text-sm text-[var(--color-text-muted)]">
					{stats.total_findings} total findings
				</div>
			</div>

			<!-- Severity Breakdown -->
			<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6">
				<div class="mb-4 text-sm text-[var(--color-text-muted)]">Severity Distribution</div>
				<div class="space-y-3">
					{#each ['critical', 'high', 'medium', 'low', 'info'] as sev}
						{@const count = stats.by_severity[sev] || 0}
						{@const pct = stats.total_findings > 0 ? (count / stats.total_findings) * 100 : 0}
						<div class="flex items-center gap-3">
							<span class="w-16 text-xs font-medium capitalize" style="color: {severityColors[sev]}">{sev}</span>
							<div class="h-2 flex-1 overflow-hidden rounded-full bg-[var(--color-bg)]">
								<div class="h-full rounded-full transition-all" style="width: {pct}%; background: {severityColors[sev]}"></div>
							</div>
							<span class="w-8 text-right text-xs text-[var(--color-text-muted)]">{count}</span>
						</div>
					{/each}
				</div>
			</div>
		</div>

		<!-- OWASP + Tool breakdown -->
		<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
			<!-- OWASP Categories -->
			<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6">
				<div class="mb-4 text-sm text-[var(--color-text-muted)]">OWASP Top 10:2025</div>
				<div class="space-y-2">
					{#each Object.entries(owaspLabels) as [cat, label]}
						{@const count = stats.by_owasp[cat] || 0}
						<button
							type="button"
							class="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-[var(--color-surface-hover)] {owaspFilter === cat ? 'bg-indigo-500/10' : ''}"
							onclick={() => owaspFilter = owaspFilter === cat ? '' : cat}
						>
							<span>
								<span class="font-mono text-[var(--color-primary)]">{cat}</span>
								<span class="ml-2 text-[var(--color-text-muted)]">{label}</span>
							</span>
							<span class="font-medium">{count}</span>
						</button>
					{/each}
				</div>
			</div>

			<!-- Tool Breakdown -->
			<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6">
				<div class="mb-4 text-sm text-[var(--color-text-muted)]">Tool Breakdown</div>
				<div class="space-y-2">
					{#each Object.entries(stats.by_tool) as [tool, count]}
						<button
							type="button"
							class="flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-[var(--color-surface-hover)] {toolFilter === tool ? 'bg-indigo-500/10' : ''}"
							onclick={() => toolFilter = toolFilter === tool ? '' : tool}
						>
							<span class="text-[var(--color-text)]">{tool}</span>
							<span class="font-medium">{count}</span>
						</button>
					{/each}
				</div>
			</div>
		</div>

		<!-- Findings Table -->
		<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)]">
			<div class="flex items-center justify-between border-b border-[var(--color-border)] px-6 py-4">
				<div class="text-sm font-medium">Findings</div>
				<div class="flex gap-2">
					<select
						bind:value={severityFilter}
						class="rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] px-3 py-1.5 text-xs"
					>
						<option value="">All Severities</option>
						{#each ['critical', 'high', 'medium', 'low', 'info'] as sev}
							<option value={sev}>{sev}</option>
						{/each}
					</select>
				</div>
			</div>

			{#if findings && findings.findings.length > 0}
				<div class="divide-y divide-[var(--color-border)]">
					{#each findings.findings as finding}
						<div class="px-6 py-3 text-sm">
							<div class="flex items-start justify-between gap-4">
								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-2">
										<span
											class="inline-block rounded px-1.5 py-0.5 text-xs font-medium capitalize"
											style="background: {severityColors[finding.severity]}20; color: {severityColors[finding.severity]}"
										>
											{finding.severity}
										</span>
										{#if finding.owasp_category}
											<span class="font-mono text-xs text-[var(--color-primary)]">
												{finding.owasp_category}
											</span>
										{/if}
										<span class="text-xs text-[var(--color-text-muted)]">{finding.tool_name}</span>
									</div>
									<div class="mt-1 text-[var(--color-text)]">{finding.message}</div>
									{#if finding.file_path}
										<div class="mt-1 font-mono text-xs text-[var(--color-text-muted)]">
											{finding.file_path}{finding.line_start ? `:${finding.line_start}` : ''}
										</div>
									{/if}
								</div>
							</div>
						</div>
					{/each}
				</div>

				{#if findings.total > findings.limit}
					<div class="border-t border-[var(--color-border)] px-6 py-3 text-center text-sm text-[var(--color-text-muted)]">
						Showing {findings.findings.length} of {findings.total} findings
					</div>
				{/if}
			{:else}
				<div class="px-6 py-8 text-center text-sm text-[var(--color-text-muted)]">
					No findings match the current filters.
				</div>
			{/if}
		</div>
	</div>
{/if}
