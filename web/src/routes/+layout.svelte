<script>
	import '../app.css';

	let { children } = $props();
</script>

<!--
  Root layout for the app.

  Why we import `app.css` here:
  - SvelteKit layouts apply to all nested routes, so global styles/variables only need to be loaded once.
  - It keeps individual pages/components focused on structure, not global theming.
-->

<div class="min-h-screen">
	<nav class="border-b border-[var(--color-border)] px-6 py-4">
		<div class="mx-auto flex max-w-7xl items-center justify-between">
			<a href="/" class="text-xl font-bold tracking-tight">
				<span class="text-[var(--color-primary)]">Secure</span>Scan
			</a>
			<div class="text-sm text-[var(--color-text-muted)]">
				OWASP Top 10:2025 Security Scanner
			</div>
		</div>
	</nav>

	<main class="mx-auto max-w-7xl px-6 py-8">
		{@render children()}
	</main>
</div>
