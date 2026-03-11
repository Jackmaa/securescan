<script lang="ts">
	/**
	 * ProgressBar is a presentational component for scan progress.
	 *
	 * Why it’s a separate component:
	 * - The scan page needs to render progress + a small event timeline.
	 * - Keeping it isolated makes it easy to restyle without touching scan orchestration logic.
	 *
	 * Data contract:
	 * - `toolsDone/toolCount` come from scan progress (SSE + polling).
	 * - `status` is the backend scan status string.
	 * - `events` is a UI-friendly list already reduced to `{ type, message }` for display.
	 */
	interface Props {
		toolsDone: number;
		toolCount: number;
		status: string;
		events: { type: string; message: string }[];
	}

	let { toolsDone, toolCount, status, events }: Props = $props();

	// Derived percentage for the progress bar width.
	// We guard toolCount=0 to avoid NaN when the backend hasn't reported totals yet.
	let progress = $derived(toolCount > 0 ? (toolsDone / toolCount) * 100 : 0);
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between text-sm">
		<span class="capitalize text-[var(--color-text)]">{status.replace(/_/g, ' ')}</span>
		{#if toolCount > 0}
			<span class="text-[var(--color-text-muted)]">{toolsDone}/{toolCount} tools</span>
		{/if}
	</div>

	<div class="h-2 overflow-hidden rounded-full bg-[var(--color-surface)]">
		<div
			class="h-full rounded-full bg-[var(--color-primary)] transition-all duration-500"
			style="width: {progress}%"
		></div>
	</div>

	<div class="max-h-64 space-y-1 overflow-y-auto">
		{#each events as event}
			<div class="flex items-start gap-2 text-sm">
				{#if event.type === 'tool_complete'}
					<span class="mt-0.5 text-[var(--color-success)]">&#10003;</span>
				{:else if event.type === 'tool_error'}
					<span class="mt-0.5 text-[var(--color-critical)]">&#10007;</span>
				{:else}
					<span class="mt-0.5 text-[var(--color-primary)]">&#8226;</span>
				{/if}
				<span class="text-[var(--color-text-muted)]">{event.message}</span>
			</div>
		{/each}
	</div>
</div>
