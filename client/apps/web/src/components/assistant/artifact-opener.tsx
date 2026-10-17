import { createContext, use, type ReactNode } from "react";

type OpenArtifact = (id: string) => void;

const NO_CARDS: ReadonlyMap<string, string> = new Map();

const ArtifactOpenerContext = createContext<OpenArtifact | null>(null);

const AttachmentCardsContext = createContext<ReadonlyMap<string, string>>(NO_CARDS);

/**
 * How the surface around a thread opens what a turn produced, for the parts
 * of a turn drawn too deep to be handed it directly — a document another
 * agent published on a task, named inside the hand-off, or the extraction
 * card read off a file, opened from the file's chip under the message that
 * carried it (attachmentCards, by document id). A surface with no pane
 * publishes nothing, and those names stay words.
 */
export function ArtifactOpenerProvider({
  onOpen,
  attachmentCards = NO_CARDS,
  children,
}: {
  onOpen?: OpenArtifact;
  attachmentCards?: ReadonlyMap<string, string>;
  children: ReactNode;
}) {
  return (
    <ArtifactOpenerContext value={onOpen ?? null}>
      <AttachmentCardsContext value={attachmentCards}>{children}</AttachmentCardsContext>
    </ArtifactOpenerContext>
  );
}

export function useArtifactOpener(): OpenArtifact | null {
  return use(ArtifactOpenerContext);
}

/** The extraction card read off a file, when the surface has one to open. */
export function useAttachmentCard(documentId: string): string | null {
  return use(AttachmentCardsContext).get(documentId) ?? null;
}
