/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { dirname, resolve } from 'path';
import { fileURLToPath } from 'url';
import { defineConfig } from 'vite';

// Get the equivalent of __dirname in ESM
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

export default defineConfig({
	plugins: [tailwindcss(), sveltekit()],
	resolve: {
		alias: {
			// Explicit aliases for Monaco Editor
			'monaco-editor': resolve(__dirname, 'node_modules/monaco-editor'),
			'monaco-themes': resolve(__dirname, 'node_modules/monaco-themes')
		}
	},
	optimizeDeps: {
		// Force include monaco-editor to optimize loading
		include: [
			'monaco-editor/esm/vs/editor/editor.api',
			'monaco-editor/esm/vs/editor/editor.worker',
			'monaco-editor/esm/vs/language/html/html.worker',
			'monaco-editor/esm/vs/language/json/json.worker',
			'monaco-themes'
		],
		esbuildOptions: {
			define: {
				global: 'globalThis' // Fix for missing global in Monaco
			}
		}
	},
	build: {
		rollupOptions: {
			output: {
				manualChunks: {
					// Create separate chunks
					monaco: ['monaco-editor/esm/vs/editor/editor.api'],
					'monaco-editor-worker': ['monaco-editor/esm/vs/editor/editor.worker'],
					'monaco-html-worker': ['monaco-editor/esm/vs/language/html/html.worker'],
					'monaco-json-worker': ['monaco-editor/esm/vs/language/json/json.worker'],
					'monaco-themes': ['monaco-themes']
				}
			}
		}
	},
	server: {
		fs: {
			// Allow serving files from the node_modules directory
			allow: ['node_modules/monaco-editor', 'node_modules/monaco-themes']
		}
	}
});
