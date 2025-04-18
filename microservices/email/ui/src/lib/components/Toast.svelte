<script lang="ts" context="module">
	// Toast types
	type ToastType = 'info' | 'success' | 'error' | 'warning';
	
	// Toast interface
	interface Toast {
		id: number;
		type: ToastType;
		message: string;
		timeout: number;
	}
	
	// Toast store
	import { writable } from 'svelte/store';
	
	// Toast default timeout
	const DEFAULT_TIMEOUT = 3000;
	
	// Create a store for toasts
	const toasts = writable<Toast[]>([]);
	
	// Toast ID counter
	let id = 0;
	
	// Toast service
	export const toast = {
		// Show a toast notification
		show: (message: string, type: ToastType = 'info', timeout: number = DEFAULT_TIMEOUT) => {
			// Create a new toast
			const newToast: Toast = {
				id: id++,
				type,
				message,
				timeout
			};
			
			// Add toast to the store
			toasts.update(all => [...all, newToast]);
			
			// Auto-remove toast after timeout
			if (timeout > 0) {
				setTimeout(() => {
					dismiss(newToast.id);
				}, timeout);
			}
			
			return newToast.id;
		},
		
		// Convenience methods
		info: (message: string, timeout = DEFAULT_TIMEOUT) => toast.show(message, 'info', timeout),
		success: (message: string, timeout = DEFAULT_TIMEOUT) => toast.show(message, 'success', timeout),
		warning: (message: string, timeout = DEFAULT_TIMEOUT) => toast.show(message, 'warning', timeout),
		error: (message: string, timeout = DEFAULT_TIMEOUT) => toast.show(message, 'error', timeout)
	};
	
	// Dismiss a toast by ID
	function dismiss(id: number) {
		toasts.update(all => all.filter(t => t.id !== id));
	}
</script>

<script lang="ts">
	// Get toasts from store
	import { fly } from 'svelte/transition';
	
	// Get icon for toast type
	function getIconForType(type: ToastType): string {
		switch (type) {
			case 'info':
				return '📋';
			case 'success':
				return '✅';
			case 'warning':
				return '⚠️';
			case 'error':
				return '❌';
			default:
				return '📋';
		}
	}
	
	// Get background color for toast type
	function getBgColorForType(type: ToastType): string {
		switch (type) {
			case 'info':
				return 'bg-blue-800/10 border-blue-700';
			case 'success':
				return 'bg-green-800/10 border-green-700';
			case 'warning':
				return 'bg-yellow-800/10 border-yellow-700';
			case 'error':
				return 'bg-red-800/10 border-red-700';
			default:
				return 'bg-zinc-800/50 border-zinc-700';
		}
	}
	
	// Get text color for toast type
	function getTextColorForType(type: ToastType): string {
		switch (type) {
			case 'info':
				return 'text-blue-200';
			case 'success':
				return 'text-green-200';
			case 'warning':
				return 'text-yellow-200';
			case 'error':
				return 'text-red-200';
			default:
				return 'text-zinc-200';
		}
	}
</script>

<div class="fixed top-4 right-4 z-50 flex flex-col gap-2 w-72">
	{#each $toasts as toast (toast.id)}
		<div
			class="p-3 border rounded-md shadow-lg {getBgColorForType(toast.type)} {getTextColorForType(toast.type)}"
			transition:fly={{ x: 20, duration: 200 }}
		>
			<div class="flex items-start">
				<div class="mr-2">{getIconForType(toast.type)}</div>
				<div class="flex-1">{toast.message}</div>
				<button
					class="ml-2 opacity-70 hover:opacity-100 focus:outline-none"
					on:click={() => dismiss(toast.id)}
				>
					✕
				</button>
			</div>
		</div>
	{/each}
</div> 