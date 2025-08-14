<script lang="ts">
	import loader from '@monaco-editor/loader';
	import type * as Monaco from 'monaco-editor/esm/vs/editor/editor.api';
	import { onDestroy, onMount } from 'svelte';

	// Component state
	let editor: Monaco.editor.IStandaloneCodeEditor | null = null;
	let monaco: typeof Monaco | null = null;
	let editorContainer: HTMLElement;
	let initializing = true;
	let monacoInitialized = false;

	// Define props with Svelte 5 syntax
	interface Props {
		value: string;
		language?: string;
		theme?: string;
		onChange?: (value: string) => void;
	}

	let {
		value = $bindable(''),
		language = 'html',
		theme = 'vs-dark',
		onChange = undefined
	}: Props = $props();

	// Initialize Monaco editor
	onMount(async () => {
		await initMonaco();
	});

	async function initMonaco() {
		try {
			// Only initialize Monaco once
			if (!monacoInitialized) {
				console.log('[TemplateEditor] Initializing Monaco');
				// Configure Monaco loader to use our bundled version
				const monacoEditor = await import('monaco-editor');
				loader.config({ monaco: monacoEditor.default });

				// Initialize Monaco
				monaco = await loader.init();

				// Set up custom themes if needed
				await registerCustomThemes(monaco);

				monacoInitialized = true;
			}

			await createEditor();
		} catch (error) {
			console.error('Failed to initialize Monaco editor:', error);
		}
	}

	// Create a new editor instance
	async function createEditor() {
		if (!monaco || !editorContainer) return;

		// Log the first 50 chars of content we're initializing with
		console.log(
			'[TemplateEditor] Creating editor with content:',
			value.substring(0, 50) + (value.length > 50 ? '...' : '')
		);

		// Dispose of previous editor if exists
		if (editor) {
			console.log('[TemplateEditor] Disposing previous editor instance');
			editor.dispose();
			editor = null;
		}

		// Setting initializing flag to prevent change events during setup
		initializing = true;

		// Create a new editor instance
		editor = monaco.editor.create(editorContainer, {
			value,
			language,
			theme,
			automaticLayout: true,
			minimap: { enabled: false },
			scrollBeyondLastLine: false,
			lineNumbers: 'on',
			glyphMargin: false,
			folding: true,
			lineDecorationsWidth: 10,
			lineNumbersMinChars: 3,
			wordWrap: 'on'
		});

		// Load saved theme preference if available
		const savedTheme = localStorage.getItem('editorTheme');
		if (savedTheme && monaco) {
			try {
				monaco.editor.setTheme(savedTheme);
			} catch (error) {
				console.error('Failed to set saved theme:', error);
			}
		}

		// Set up content change handler
		editor.onDidChangeModelContent(
			debounce((e: Monaco.editor.IModelContentChangedEvent) => {
				if (e.isFlush) {
					// Ignore setValue calls
				} else {
					// User input - update bound value and call onChange
					const newValue = editor?.getValue() ?? '';

					// Only update if not initializing and value has changed
					if (!initializing && value !== newValue) {
						console.log('[TemplateEditor] Content changed by user, updating');
						value = newValue;
						// Call the onChange prop if provided
						if (onChange) onChange(newValue);
					}
				}
			}, 100)
		);

		// Mark initialization as complete
		initializing = false;
		console.log('[TemplateEditor] Editor initialization complete');
	}

	// Watch for value changes from outside - recreate editor for significant changes
	$effect(() => {
		if (!monacoInitialized) return;

		// Check if we have a significant content change (more than just typing)
		// This is a heuristic to detect template switches vs normal typing
		const hasSignificantChange =
			!editor || Math.abs(value.length - (editor.getValue()?.length || 0)) > 50;

		if (hasSignificantChange) {
			console.log('[TemplateEditor] Significant content change detected, recreating editor');
			createEditor();
		} else if (editor && !initializing) {
			const currentValue = editor.getValue();
			if (value !== currentValue) {
				console.log('[TemplateEditor] Minor content update, setting value directly');
				// Temporarily set initializing to true to prevent circular updates
				initializing = true;
				editor.setValue(value);
				initializing = false;
			}
		}
	});

	// Register custom themes for Monaco
	async function registerCustomThemes(monaco: typeof Monaco): Promise<void> {
		try {
			// Brilliance Black theme
			const brillianceBlack = (await import('monaco-themes/themes/Brilliance Black.json')).default;
			monaco.editor.defineTheme(
				'brilliance-black',
				brillianceBlack as Monaco.editor.IStandaloneThemeData
			);

			// Other themes
			const dracula = (await import('monaco-themes/themes/Dracula.json')).default;
			monaco.editor.defineTheme('dracula', dracula as Monaco.editor.IStandaloneThemeData);

			const monokai = (await import('monaco-themes/themes/Monokai.json')).default;
			monaco.editor.defineTheme('monokai', monokai as Monaco.editor.IStandaloneThemeData);

			const github = (await import('monaco-themes/themes/GitHub.json')).default;
			monaco.editor.defineTheme('github', github as Monaco.editor.IStandaloneThemeData);

			const solarizedDark = (await import('monaco-themes/themes/Solarized-dark.json')).default;
			monaco.editor.defineTheme(
				'solarized-dark',
				solarizedDark as Monaco.editor.IStandaloneThemeData
			);

			const nord = (await import('monaco-themes/themes/Nord.json')).default;
			monaco.editor.defineTheme('nord', nord as Monaco.editor.IStandaloneThemeData);
		} catch (error) {
			console.error('Failed to load Monaco themes:', error);
		}
	}

	// Clean up on destroy
	onDestroy(() => {
		if (editor) {
			editor.dispose();
		}
		if (monaco) {
			monaco.editor.getModels().forEach((model) => model.dispose());
		}
	});

	// Utility: Debounce function to limit frequent calls
	function debounce(func: Function, wait: number) {
		let timeout: ReturnType<typeof setTimeout>;
		return function executedFunction(...args: any[]) {
			clearTimeout(timeout);
			timeout = setTimeout(() => func(...args), wait);
		};
	}

	// Handle theme change in Svelte 5 style
	function handleThemeChange(e: Event) {
		const target = e.target as HTMLSelectElement;
		const newTheme = target.value;
		if (monaco && editor) {
			monaco.editor.setTheme(newTheme);
			localStorage.setItem('editorTheme', newTheme);
		}
	}
</script>

<div class="flex w-1/2 flex-col overflow-hidden border-r border-zinc-800">
	<div class="flex items-center justify-between border-b border-zinc-800 bg-zinc-950 px-4 py-2">
		<h3 class="text-sm font-medium text-zinc-200">Template Code</h3>
		<div class="flex items-center space-x-2">
			<select
				id="theme-selector"
				class="rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-xs text-zinc-200 focus:ring-1 focus:ring-zinc-600 focus:outline-none"
				onchange={handleThemeChange}
				value={theme}
			>
				<option value="brilliance-black">Brilliance Black</option>
				<option value="dracula">Dracula</option>
				<option value="monokai">Monokai</option>
				<option value="github">GitHub</option>
				<option value="solarized-dark">Solarized Dark</option>
				<option value="nord">Nord</option>
			</select>
		</div>
	</div>
	<div bind:this={editorContainer} class="flex-1"></div>
</div>
