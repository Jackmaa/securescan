<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { getScan } from '$lib/api';
	import { connectSSE } from '$lib/sse';
	import ProgressBar from '$lib/components/ProgressBar.svelte';
	import type { SSEEvent } from '$lib/types';

	let scanId = $derived($page.params.id);
	let status = $state('pending');
	let toolCount = $state(0);
	let toolsDone = $state(0);
	let events = $state<{ type: string; message: string }[]>([]);
	let sse: ReturnType<typeof connectSSE> | null = null;

	function addEvent(type: string, message: string) {
		events = [...events, { type, message }];
	}

	function handleSSEEvent(event: SSEEvent) {
		const d = event.data as Record<string, unknown>;

		switch (event.type) {
			case 'status':
				status = (d.status as string) || status;
				addEvent('status', (d.message as string) || status);
				break;
			case 'tool_start':
				toolCount = (d.total as number) || toolCount;
				addEvent('tool_start', `Running ${d.tool}...`);
				break;
			case 'tool_complete':
				toolsDone++;
				addEvent('tool_complete', `${d.tool}: ${d.findings_count} findings`);
				break;
			case 'tool_error':
				addEvent('tool_error', `${d.tool}: ${d.error}`);
				break;
			case 'mapping':
			case 'scoring':
				status = event.type;
				addEvent('status', d.message as string);
				break;
			case 'complete':
				status = 'completed';
				addEvent('status', 'Scan complete!');
				sse?.close();
				// Navigate to dashboard after a brief pause so the user sees the completion
				setTimeout(() => goto(`/scan/${scanId}/dashboard`), 1500);
				break;
		}
	}

	$effect(() => {
		sse = connectSSE(scanId, handleSSEEvent);

		// Also poll the scan object to sync state on page load
		getScan(scanId).then((scan) => {
			status = scan.status;
			toolCount = scan.tool_count;
			toolsDone = scan.tools_done;
			if (scan.status === 'completed') {
				sse?.close();
				goto(`/scan/${scanId}/dashboard`);
			}
		});

		return () => sse?.close();
	});
</script>

<svelte:head>
	<title>Scanning... | SecureScan</title>
</svelte:head>

<div class="mx-auto max-w-2xl py-16">
	<h1 class="mb-8 text-center text-2xl font-bold">Security Scan in Progress</h1>

	<div class="rounded-xl border border-[var(--color-border)] bg-[var(--color-surface)] p-6">
		<ProgressBar {toolsDone} {toolCount} {status} {events} />
	</div>

	{#if status === 'failed'}
		<div class="mt-4 rounded-lg border border-[var(--color-critical)]/30 bg-[var(--color-critical)]/10 px-4 py-3 text-sm text-[var(--color-critical)]">
			Scan failed. Check the API logs for details.
		</div>
	{/if}
</div>
