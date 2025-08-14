<script lang="ts">
	import { toaster } from '$lib/toaster';
	import { Toaster } from '@skeletonlabs/skeleton-svelte';
	import '../app.css';
	let { children } = $props();
</script>

{@render children()}

<Toaster {toaster} />