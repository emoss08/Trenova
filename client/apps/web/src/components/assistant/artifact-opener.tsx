import type { BoardCallTool } from "@/lib/board-place";
import type { AssistantArtifact } from "@/types/assistant";
import { createContext, use, type ComponentType, type ReactElement, type ReactNode } from "react";

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

/**
 * A board a reply's turn placed in the conversation: the artifact as the
 * thread lists it, and the arguments of the call that made it (a show_board
 * call's are the payload's `board`; a show_answer call's are its answer and
 * groups), which the reply draws from while the board is read. `spec` is null
 * when that call is not in view.
 */
export type ReplyBoard = {
  artifact: AssistantArtifact;
  /** The tool whose call made it, which says what shape `spec` has. */
  tool: BoardCallTool;
  spec: Record<string, unknown> | null;
};

export type ReplyBoardsProps = {
  /** The boards the reply draws, in the order its turn showed them. */
  boards: readonly ReplyBoard[];
  /** The reply's own words: what is drawn when the boards cannot be read; null when it has none. */
  fallback: ReactElement | null;
  /** When the reply was saved, in Unix seconds. */
  answeredAt: number;
  /**
   * The reply landed while the person was watching, so it arrives rather than
   * appears. Settled by the reply when it mounts; it never turns true later.
   */
  arrives: boolean;
  /**
   * Told, before the frame is painted, once every board is drawn or could not
   * be read, so what stands under the reply (its actions) comes with the
   * boards and never under their skeleton.
   */
  onDrawn?: () => void;
};

/** A board the running turn is still showing: the arguments of the call making it. */
export type PendingBoardProps = { tool: BoardCallTool; spec: Record<string, unknown> | null };

/** How a surface draws the boards a reply placed in the conversation. */
export type ReplyBoardRenderer = {
  Boards: ComponentType<ReplyBoardsProps>;
  Pending: ComponentType<PendingBoardProps>;
};

const ReplyBoardsContext = createContext<ReplyBoardRenderer | null>(null);

/**
 * Lets the replies under it draw the boards their turns placed in the
 * conversation instead of their words. Only the Desk's conversation supplies
 * one; a surface that supplies none (the page assistant, the assistant panel,
 * the inbox hand-off, the margin) draws every reply's words, which already
 * say the answer and link the board.
 */
export function ReplyBoardsProvider({
  renderer,
  children,
}: {
  renderer: ReplyBoardRenderer;
  children: ReactNode;
}) {
  return <ReplyBoardsContext value={renderer}>{children}</ReplyBoardsContext>;
}

export function useReplyBoards(): ReplyBoardRenderer | null {
  return use(ReplyBoardsContext);
}
