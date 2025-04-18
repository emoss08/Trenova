<script lang="ts">
	import { onMount } from 'svelte';

	// Define props with Svelte 5 syntax
	interface Props {
		currentTemplate: string | null;
		previewContent: string;
		onrefresh?: () => void;
	}

	let {
		currentTemplate = $bindable(null),
		previewContent = $bindable(''),
		onrefresh = undefined
	}: Props = $props();

	// Export const for external reference instead of unused property
	export const previewTemplate = null; // Just for type compatibility

	let previewFrame: HTMLIFrameElement;
	let previousTemplate: string | null = null;

	// Watch previewContent for changes
	$effect(() => {
		if (previewContent && previewFrame) {
			console.log(`Content changed, updating preview frame: ${previewContent.length} bytes`);
			updatePreviewFrame();
		}
	});

	// Watch template changes
	$effect(() => {
		if (currentTemplate && currentTemplate !== previousTemplate) {
			console.log(
				`[DIRECT WATCH] Template changed from '${previousTemplate}' to '${currentTemplate}'`
			);
			refreshPreview();
			previousTemplate = currentTemplate;
		}
	});

	// Refresh preview on demand - use the onrefresh prop
	function refreshPreview(): void {
		console.log('Manual refresh requested');
		if (onrefresh) onrefresh();
	}

	// Update the iframe content
	function updatePreviewFrame(): void {
		if (!previewFrame) {
			console.warn('Preview frame not available yet');
			return;
		}

		console.log('Updating preview frame with content');
		const doc = previewFrame.contentDocument || previewFrame.contentWindow?.document;
		if (!doc) {
			console.warn('Could not access iframe document');
			return;
		}

		try {
			doc.open();
			doc.write(previewContent);
			doc.close();
			console.log('Preview frame updated successfully');
		} catch (error) {
			console.error('Error updating preview frame:', error);
		}
	}

	// Initialize preview after component mounts
	onMount(() => {
		console.log('TemplatePreview mounted, currentTemplate:', currentTemplate);
		if (currentTemplate) {
			refreshPreview();
		}
	});
</script>

<div class="flex w-1/2 flex-col overflow-hidden">
	<div class="flex items-center justify-between border-b border-zinc-800 bg-zinc-950 px-4 py-2">
		<h3 class="text-sm font-medium text-zinc-200">Preview ({currentTemplate || 'none'})</h3>
		<button
			class="rounded bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-700 transition-colors hover:bg-zinc-200 focus:ring-2 focus:ring-zinc-300 focus:ring-offset-1 focus:outline-none"
			onclick={refreshPreview}
			disabled={!currentTemplate}
		>
			Refresh
		</button>
	</div>
	<div class="flex-1 overflow-auto bg-white">
		<iframe bind:this={previewFrame} title="Template Preview" class="h-full w-full border-none"
		></iframe>
	</div>
</div>
