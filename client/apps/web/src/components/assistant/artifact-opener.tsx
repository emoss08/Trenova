import { createContext, use, type ReactNode } from "react";

type OpenArtifact = (id: string) => void;

const ArtifactOpenerContext = createContext<OpenArtifact | null>(null);

/**
 * How the surface around a thread opens what a turn produced, for the parts
 * of a turn drawn too deep to be handed it directly — a document another
 * agent published on a task, named inside the hand-off. A surface with no
 * pane publishes nothing, and those names stay words.
 */
export function ArtifactOpenerProvider({
  onOpen,
  children,
}: {
  onOpen?: OpenArtifact;
  children: ReactNode;
}) {
  return <ArtifactOpenerContext value={onOpen ?? null}>{children}</ArtifactOpenerContext>;
}

export function useArtifactOpener(): OpenArtifact | null {
  return use(ArtifactOpenerContext);
}
