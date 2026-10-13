/** What vite/service-worker.ts writes into sw.js about the build it belongs to. */
export type ShellManifest = {
  /** Identifies the build; names its shell cache. Changes whenever any built file does. */
  version: string;
  /** The app's mount path, ending in "/". */
  base: string;
  /** The build's entry script; the worker only caches HTML that loads it. */
  entry: string;
  /** The files the HTML shell loads before it can render, fetched when the worker installs. */
  startup: string[];
  /** Every build file under assets/, so the asset cache keeps only this build's. */
  assets: string[];
};

/**
 * Whether HTML is the shell of the manifest's build. While a release rolls out, a CDN can
 * briefly answer with the previous build's index.html; a worker that cached that would open
 * the previous build for the whole of the release.
 */
export function shellLoadsBuild(html: string, manifest: Pick<ShellManifest, "entry">): boolean {
  return html.includes(`"${manifest.entry}"`);
}
