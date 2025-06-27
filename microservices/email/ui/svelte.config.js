import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { mdsvex } from 'mdsvex';

const config = {
	preprocess: [vitePreprocess(), mdsvex()],
	kit: {
		adapter: adapter({
			// Output to build directory for the Go server to serve
			pages: 'build',
			assets: 'build',
			fallback: 'index.html',
			precompress: false
		})
	},
	alias: {
		'@/*': './src/lib/*'
	},
	extensions: ['.svelte', '.svx']
};

export default config;
