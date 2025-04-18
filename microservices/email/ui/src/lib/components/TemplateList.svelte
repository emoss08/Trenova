<script lang="ts">
	export let templates: string[] = [];
	export let currentTemplate: string | null = null;
	export let onSelectTemplate: (name: string) => void = () => {};
	
	// Handle template selection with added debugging
	function handleTemplateSelection(template: string) {
		console.log(`[TemplateList] Template selected: ${template}`);
		if (template !== currentTemplate) {
			console.log(`[TemplateList] Calling onSelectTemplate for: ${template}`);
			onSelectTemplate(template);
		} else {
			console.log(`[TemplateList] Template ${template} already selected, not calling onSelectTemplate`);
		}
	}
</script>

<div class="flex w-64 flex-col border-r border-zinc-800 bg-zinc-950">
	<div class="border-b border-zinc-800 p-4">
		<h2 class="text-lg font-semibold text-white">Templates</h2>
	</div>

	<div class="flex-1 overflow-y-auto p-3">
		{#if templates.length === 0}
			<div class="animate-pulse space-y-2">
				<div class="h-10 rounded bg-zinc-800"></div>
				<div class="h-10 rounded bg-zinc-800"></div>
				<div class="h-10 rounded bg-zinc-800"></div>
			</div>
		{:else}
			<ul class="space-y-1">
				{#each templates as template}
					<li>
						<button
							class="w-full rounded px-3 py-2 text-left text-sm font-medium transition-colors {currentTemplate ===
							template
								? 'bg-zinc-800 text-zinc-200'
								: 'text-zinc-400 hover:bg-zinc-800/70 hover:text-zinc-300'}"
							on:click={() => handleTemplateSelection(template)}
						>
							{template}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
</div> 