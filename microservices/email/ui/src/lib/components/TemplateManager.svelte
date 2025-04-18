<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import SampleDataEditor from './SampleDataEditor.svelte';
	import TemplateEditor from './TemplateEditor.svelte';
	import TemplateList from './TemplateList.svelte';
	import TemplatePreview from './TemplatePreview.svelte';
	import { toast } from './Toast.svelte';

	// State
	let templates: string[] = [];
	let currentTemplate: string | null = null;
	let templateContent: string = '';
	let sampleData: Record<string, any> = {};
	let activeTab: 'editor' | 'samples' = 'editor';
	let socket: WebSocket | null = null;
	let previewHtml: string = ''; // Store the preview HTML directly
	let livePreviewEnabled: boolean = true; // Enable live preview by default
	let previewDebounceTimer: ReturnType<typeof setTimeout> | null = null;
	let showEditor = true;

	// Define WebSocket message type
	interface WebSocketMessage {
		type: string;
		templateName?: string;
		sampleName?: string;
	}

	// Load templates on mount
	onMount(() => {
		fetchTemplates().then(() => {
			// If we have a template selected and we're in editor mode,
			// make sure to load the preview immediately
			if (currentTemplate && activeTab === 'editor') {
				console.log('Initial mount with template, updating preview');
				updatePreview();
			}
		});
		setupWebSocket();
	});

	// Clean up on destroy
	onDestroy(() => {
		if (socket) {
			socket.close();
		}

		// Clear any pending debounce timers
		if (previewDebounceTimer) {
			clearTimeout(previewDebounceTimer);
		}
	});

	// Setup WebSocket for real-time updates
	function setupWebSocket(): void {
		const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
		const wsUrl = `${protocol}//${window.location.host}/ws`;

		socket = new WebSocket(wsUrl);

		socket.addEventListener('open', () => {
			console.log('Connected to WebSocket server');
		});

		socket.addEventListener('message', (event) => {
			try {
				const data = JSON.parse(event.data) as WebSocketMessage;

				if (data.type === 'template_updated' && data.templateName === currentTemplate) {
					toast.info(`Template "${data.templateName}" was updated by another user`);
					fetchTemplateContent(currentTemplate);
				} else if (data.type === 'sample_updated' && data.sampleName === currentTemplate) {
					toast.info(`Sample data for "${data.sampleName}" was updated by another user`);
					if (activeTab === 'samples') {
						fetchSampleData(currentTemplate);
					}
				}
			} catch (error) {
				console.error('Error processing WebSocket message:', error);
			}
		});

		socket.addEventListener('close', () => {
			console.log('Disconnected from WebSocket server');
			// Try to reconnect after 3 seconds
			setTimeout(setupWebSocket, 3000);
		});
	}

	// Fetch all templates
	async function fetchTemplates(): Promise<void> {
		try {
			const response = await fetch('http://localhost:3002/api/templates');
			templates = await response.json();

			// Select first template by default if we don't have one selected
			if (templates.length > 0 && !currentTemplate) {
				console.log('[fetchTemplates] No template selected, selecting first template:', templates[0]);
				// Force set to editor tab
				activeTab = 'editor';
				
				// Directly select the template (which will fetch content and update preview)
				await selectTemplate(templates[0]);
			}
		} catch (error) {
			console.error('Failed to fetch templates:', error);
			toast.error('Failed to load templates');
		}
	}

	// Select a template and load its content
	async function selectTemplate(name: string): Promise<void> {
		console.log(`Selecting template: ${name}`);
		
		try {
			// Fetch the template content
			const response = await fetch(`http://localhost:3002/api/templates/${name}`);
			const content = await response.text();
			console.log(`[selectTemplate] Fetched template content: ${name}, length: ${content.length}`);
			
			// First update the current template name
			currentTemplate = name;
			
			// Hide the editor during the update
			showEditor = false;
			
			// Reset template content
			templateContent = '';
			
			// Update content and show editor in the next frame
			setTimeout(() => {
				// Set the actual content
				console.log(`[selectTemplate] Setting template content for: ${name}`);
				templateContent = content;
				
				// Show the editor again
				showEditor = true;
				
				// Force an immediate preview refresh
				console.log(`[selectTemplate] Updating preview for: ${name}`);
				updatePreview();
			}, 150); // Give more time for DOM updates to process
			
			// If we're in samples tab, load the sample data as well
			if (activeTab === 'samples') {
				await fetchSampleData(name);
			}
		} catch (error) {
			console.error(`Failed to select template ${name}:`, error);
			toast.error(`Failed to select template ${name}`);
		}
	}

	// Fetch template content
	async function fetchTemplateContent(name: string): Promise<void> {
		try {
			console.log(`[fetchTemplateContent] Fetching content for template: ${name}`);
			const response = await fetch(`http://localhost:3002/api/templates/${name}`);
			const content = await response.text();
			console.log(`[fetchTemplateContent] Received template content for ${name}, ${content.length} bytes`);
			// Force update the templateContent to trigger reactivity
			templateContent = content;
		} catch (error) {
			console.error(`Failed to fetch template ${name}:`, error);
			toast.error(`Failed to load template ${name}`);
		}
	}

	// Fetch sample data for a template
	async function fetchSampleData(name: string): Promise<void> {
		try {
			const response = await fetch(`http://localhost:3002/api/samples/${name}`);
			sampleData = await response.json();
		} catch (error) {
			console.error(`Failed to fetch sample data for ${name}:`, error);
			toast.error(`Failed to load sample data for ${name}`);
		}
	}

	// Save template content
	async function saveTemplate(): Promise<void> {
		if (!currentTemplate) return;

		try {
			const response = await fetch(`http://localhost:3002/api/templates/${currentTemplate}`, {
				method: 'PUT',
				body: templateContent
			});

			if (response.ok) {
				toast.success(`Template ${currentTemplate} saved successfully`);
			} else {
				throw new Error(`Server returned ${response.status}`);
			}
		} catch (error) {
			console.error(`Failed to save template ${currentTemplate}:`, error);
			toast.error(`Failed to save template ${currentTemplate}`);
		}
	}

	// Save sample data
	async function saveSampleData(): Promise<void> {
		if (!currentTemplate) return;

		try {
			const response = await fetch(`http://localhost:3002/api/samples/${currentTemplate}`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(sampleData)
			});

			if (response.ok) {
				toast.success(`Sample data for ${currentTemplate} saved successfully`);
			} else {
				throw new Error(`Server returned ${response.status}`);
			}
		} catch (error) {
			console.error(`Failed to save sample data for ${currentTemplate}:`, error);
			toast.error(`Failed to save sample data for ${currentTemplate}`);
		}
	}

	// Handle template content changes with debouncing for live preview
	function handleEditorChange(value: string): void {
		// Update the content immediately
		templateContent = value;

		// Cancel any pending debounce timer
		if (previewDebounceTimer) {
			clearTimeout(previewDebounceTimer);
		}

		// If live preview is enabled, update the preview after a short delay
		if (livePreviewEnabled) {
			previewDebounceTimer = setTimeout(() => {
				console.log('Content changed, updating preview (debounced)');
				updatePreview();
			}, 500); // 500ms debounce
		}
	}

	// Toggle live preview
	function toggleLivePreview(): void {
		livePreviewEnabled = !livePreviewEnabled;
		toast.info(livePreviewEnabled ? 'Live preview enabled' : 'Live preview disabled');
	}

	// Update the preview immediately
	async function updatePreview(): Promise<void> {
		if (!currentTemplate) return;

		try {
			console.log(`Updating preview for template: ${currentTemplate}`);
			// Get the preview HTML
			previewHtml = await previewTemplate();
			console.log(`Preview updated, content length: ${previewHtml.length}`);
		} catch (error) {
			console.error('Failed to update preview:', error);
		}
	}

	// Preview template
	async function previewTemplate(): Promise<string> {
		if (!currentTemplate) return '';

		try {
			console.log(`Sending preview request for template: ${currentTemplate}`);
			const response = await fetch(
				`http://localhost:3002/api/templates/preview/${currentTemplate}`,
				{
					method: 'POST',
					headers: {
						'Content-Type': 'text/plain'
					},
					body: templateContent
				}
			);

			const html = await response.text();
			console.log(`Received preview HTML for ${currentTemplate}, ${html.length} bytes`);
			return html;
		} catch (error) {
			console.error(`Failed to preview template ${currentTemplate}:`, error);
			toast.error(`Failed to preview template ${currentTemplate}`);
			return '';
		}
	}

	// Switch tabs
	function setTab(tab: 'editor' | 'samples'): void {
		activeTab = tab;
		if (tab === 'samples' && currentTemplate) {
			fetchSampleData(currentTemplate);
		}
	}
</script>

<div class="flex h-screen flex-col">
	<header class="border-b border-zinc-800 bg-zinc-950 py-3 shadow-sm">
		<div class="flex items-center justify-between px-4">
			<div class="flex items-center space-x-2">
				<h1 class="text-left text-xl font-semibold text-white">Email Template Manager</h1>
			</div>
			<div
				class="bg-primary-700/20 text-primary-200 border-primary-700/20 rounded-full border px-3 py-1 text-sm"
			>
				Development Mode
			</div>
		</div>
	</header>

	<main class="flex flex-1 overflow-hidden">
		<!-- Sidebar -->
		<TemplateList {templates} {currentTemplate} onSelectTemplate={selectTemplate} />

		<!-- Main Content -->
		<div class="flex flex-1 flex-col overflow-hidden">
			<!-- Tabs -->
			<div class="border-b border-zinc-800 bg-zinc-950">
				<div class="flex">
					<button
						class="whitespace-nowrap border-b-2 px-6 py-4 text-sm font-medium {activeTab ===
						'editor'
							? 'border-zinc-500 text-zinc-200'
							: 'border-transparent text-zinc-500 hover:border-zinc-400 hover:text-zinc-400'}"
						on:click={() => setTab('editor')}
					>
						Template Editor
					</button>
					<button
						class="whitespace-nowrap border-b-2 px-6 py-4 text-sm font-medium {activeTab ===
						'samples'
							? 'border-zinc-500 text-zinc-200'
							: 'border-transparent text-zinc-500 hover:border-zinc-400 hover:text-zinc-400'}"
						on:click={() => setTab('samples')}
					>
						Sample Data
					</button>
				</div>
			</div>

			<!-- Editor Section -->
			{#if activeTab === 'editor'}
				<div class="flex flex-1 flex-col overflow-hidden">
					<div
						class="flex items-center justify-between border-b border-zinc-800 bg-zinc-950 px-4 py-3"
					>
						<h2 class="text-lg font-medium text-zinc-200">
							{currentTemplate || 'Select a template'}
						</h2>
						<div class="flex gap-2">
							<button
								class="rounded bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-700 transition-colors hover:bg-zinc-200 focus:outline-none focus:ring-2 focus:ring-zinc-300 focus:ring-offset-1"
								on:click={toggleLivePreview}
							>
								{livePreviewEnabled ? 'Disable Live Preview' : 'Enable Live Preview'}
							</button>
							<button
								class="rounded bg-zinc-100 px-3 py-1 text-xs font-medium text-zinc-700 transition-colors hover:bg-zinc-200 focus:outline-none focus:ring-2 focus:ring-zinc-300 focus:ring-offset-1"
								on:click={updatePreview}
								disabled={!currentTemplate}
							>
								Force Refresh Preview
							</button>
							<button
								class="rounded-md bg-zinc-700/50 px-4 py-2 text-sm font-medium text-zinc-200 shadow-sm transition-colors hover:bg-zinc-700/80 focus:outline-none focus:ring-2 focus:ring-zinc-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
								on:click={saveTemplate}
								disabled={!currentTemplate}
							>
								Save
							</button>
						</div>
					</div>

					<div class="flex flex-1 overflow-hidden">
						{#if showEditor}
							<TemplateEditor
								value={templateContent}
								language="html"
								theme="brilliance-black"
								onChange={handleEditorChange}
							/>
						{:else}
							<div class="flex w-1/2 flex-col overflow-hidden border-r border-zinc-800">
								<div class="flex items-center justify-between border-b border-zinc-800 bg-zinc-950 px-4 py-2">
									<h3 class="text-sm font-medium text-zinc-200">Template Code</h3>
									<div class="animate-pulse h-5 w-32 bg-zinc-800 rounded"></div>
								</div>
								<div class="flex-1 bg-zinc-900 flex items-center justify-center">
									<div class="text-zinc-500">Loading template...</div>
								</div>
							</div>
						{/if}
						<TemplatePreview
							{currentTemplate}
							previewContent={previewHtml}
							onrefresh={updatePreview}
						/>
					</div>
				</div>
			{:else}
				<div class="flex flex-1 flex-col overflow-hidden">
					<div
						class="flex items-center justify-between border-b border-zinc-800 bg-zinc-950 px-4 py-3"
					>
						<h2 class="text-lg font-medium text-zinc-200">Sample Data</h2>
						<button
							class="rounded-md bg-zinc-700/50 px-4 py-2 text-sm font-medium text-zinc-200 shadow-sm transition-colors hover:bg-zinc-700/80 focus:outline-none focus:ring-2 focus:ring-zinc-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
							on:click={saveSampleData}
							disabled={!currentTemplate}
						>
							Save
						</button>
					</div>
					<SampleDataEditor 
						value={sampleData} 
						on:change={(e: { detail: Record<string, any> }) => (sampleData = e.detail)} 
					/>
				</div>
			{/if}
		</div>
	</main>
</div> 