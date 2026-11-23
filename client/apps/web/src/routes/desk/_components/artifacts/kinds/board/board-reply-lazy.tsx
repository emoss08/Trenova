import type {
  PendingBoardProps,
  ReplyBoardRenderer,
  ReplyBoardsProps,
} from "@/components/assistant/artifact-opener";
import { lazyComponent, preloadable } from "@/lib/lazy-component";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { BoardWaitRow } from "./board-wait-row";

type BoardReplyModule = typeof import("./board-reply");

/** The drawing's code once it has arrived, so a reply drawn after that waits for nothing. */
let loaded: BoardReplyModule | null = null;

/** The conversation's board drawing, read with the board's parts the first time a reply needs it. */
export const preloadReplyBoards = preloadable(() =>
  import("./board-reply").then((module) => {
    loaded = module;
    return module;
  }),
);

const boards = lazyComponent(preloadReplyBoards, (module) => module.DeskReplyBoards);
const pending = lazyComponent(preloadReplyBoards, (module) => module.DeskPendingBoard);

/**
 * What stands while the drawing's code is read: the row the reply's actions
 * will take, saying the board is being read, never the reply's words, which
 * would show for a moment and then give way to the board.
 */
function BoardCodeWait() {
  const t = useT();
  return (
    <div aria-busy="true" aria-label={t("Reading the board…")}>
      <BoardWaitRow />
    </div>
  );
}

function LazyReplyBoards(props: ReplyBoardsProps) {
  // A lazy component first drawn after its code arrived still waits a
  // microtask, which paints its fallback for a frame; once the code is here
  // the drawing is used directly. Settled when the reply mounts, so it never
  // swaps mid-life.
  const [ready] = useState(() => loaded);
  if (ready) return <ready.DeskReplyBoards {...props} />;
  return (
    <LazyComponent fallback={<BoardCodeWait />}>
      <boards.Component {...props} />
    </LazyComponent>
  );
}

function LazyPendingBoard(props: PendingBoardProps) {
  // The pending board is itself a skeleton, and its code is read while the
  // show_board call runs on the server, well before the call finishes.
  const [ready] = useState(() => loaded);
  if (ready) return <ready.DeskPendingBoard {...props} />;
  return (
    <LazyComponent fallback={null}>
      <pending.Component {...props} />
    </LazyComponent>
  );
}

/** How the Desk's conversation draws the boards its replies placed there. */
export const DESK_REPLY_BOARDS: ReplyBoardRenderer = {
  Boards: LazyReplyBoards,
  Pending: LazyPendingBoard,
};
