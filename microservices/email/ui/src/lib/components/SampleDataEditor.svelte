<script lang="ts">
	import { createEventDispatcher, onMount } from 'svelte';

	// Define props
	interface Props {
		value: Record<string, any>;
	}

	let { value = $bindable({}) }: Props = $props();
	
	// Since createEventDispatcher is deprecated in Svelte 5, we'll keep it for 
	// backward compatibility but also provide a modern approach
	const dispatch = createEventDispatcher<{ change: Record<string, any> }>();
	
	// Local editor state
	let editor: HTMLTextAreaElement;
	let editorContent = '';
	let error: string | null = null;
	
	// Format JSON with indentation for the editor
	function formatJson(obj: Record<string, any>): string {
		try {
			return JSON.stringify(obj, null, 2);
		} catch (e) {
			console.error('Error formatting JSON:', e);
			return '{}';
		}
	}
	
	// Parse JSON from the editor
	function parseJson(text: string): Record<string, any> | null {
		try {
			// If the text is empty, return an empty object
			if (!text.trim()) {
				return {};
			}
			return JSON.parse(text);
		} catch (e) {
			console.error('Error parsing JSON:', e);
			return null;
		}
	}
	
	// Update editor content when the value changes
	$effect(() => {
		editorContent = formatJson(value);
	});
	
	// Handle content changes in the editor
	function handleEditorChange() {
		const parsed = parseJson(editorContent);
		
		if (parsed === null) {
			error = 'Invalid JSON format';
		} else {
			error = null;
			
			// Avoid unnecessary updates
			if (JSON.stringify(parsed) !== JSON.stringify(value)) {
				// Update value using Svelte 5 reactivity
				value = parsed;
				
				// Also dispatch event for backward compatibility
				dispatch('change', parsed);
			}
		}
	}
	
	// Initialize the editor with formatted JSON on mount
	onMount(() => {
		editorContent = formatJson(value);
	});
</script>

<div class="flex flex-col h-full">
	<div class="flex-1 p-4 overflow-auto">
		<textarea
			bind:this={editor}
			bind:value={editorContent}
			on:input={handleEditorChange}
			class="w-full h-full p-3 font-mono text-sm bg-zinc-900 text-zinc-200 rounded border border-zinc-700 focus:outline-none focus:ring-1 focus:ring-zinc-500"
			placeholder="Enter sample JSON data..."
		></textarea>
	</div>
	
	{#if error}
		<div class="p-2 m-2 text-sm text-red-400 bg-red-900/20 border border-red-800 rounded">
			{error}
		</div>
	{/if}
</div> 