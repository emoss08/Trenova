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
	let editorContent = $state('');
	let error = $state<string | null>(null);

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

<div class="flex h-full flex-col">
	<div class="flex-1 overflow-auto p-4">
		<textarea
			bind:this={editor}
			bind:value={editorContent}
			oninput={handleEditorChange}
			class="h-full w-full rounded border border-zinc-700 bg-zinc-900 p-3 font-mono text-sm text-zinc-200 focus:ring-1 focus:ring-zinc-500 focus:outline-none"
			placeholder="Enter sample JSON data..."
		></textarea>
	</div>

	{#if error}
		<div class="m-2 rounded border border-red-800 bg-red-900/20 p-2 text-sm text-red-400">
			{error}
		</div>
	{/if}
</div>
