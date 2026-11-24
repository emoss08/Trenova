import { createContext, type ReactNode } from "react";

/**
 * Draws an artifact a reply or a board's words name, by the artifact's own
 * title, or returns null when the surface does not know that artifact, which
 * leaves the link its words. Its own module, so the board's words read it
 * without loading the markdown renderer.
 */
export type ArtifactLinkRenderer = (id: string) => ReactNode | null;

export const ArtifactLinkContext = createContext<ArtifactLinkRenderer | null>(null);
