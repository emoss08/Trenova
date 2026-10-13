import { startupFiles } from "@/lib/startup-files";

/**
 * Which build of the app this page is running. In a production build it is the entry
 * script's URL, whose content hash changes whenever any code the app starts with changes;
 * the dev server's entry never changes, so it reads "development".
 *
 * Read from the page rather than injected at build time so that two builds of the same
 * source produce the same files.
 */
export function buildId(doc: Pick<Document, "querySelector"> = document): string {
  const entry = doc.querySelector<HTMLScriptElement>('script[type="module"][src]');
  const src = entry?.getAttribute("src");
  return src && !src.startsWith("/src/") ? src : "development";
}

/**
 * Which build the server is deploying now, read from its index.html past every cache (the
 * service worker leaves a plain fetch to the network). Null when the server cannot be asked
 * (offline) or answers with something that is not the app.
 */
export async function deployedBuildId(
  fetchPage: (url: string, init: RequestInit) => Promise<Response> = (url, init) =>
    fetch(url, init),
  base: string = import.meta.env.BASE_URL,
): Promise<string | null> {
  try {
    const response = await fetchPage(base, { cache: "no-store", credentials: "same-origin" });
    if (!response.ok) return null;
    return startupFiles(await response.text()).entry;
  } catch {
    return null;
  }
}
