import {
  useArtifactOpener,
  type PendingBoardProps,
  type ReplyBoard,
  type ReplyBoardsProps,
} from "@/components/assistant/artifact-opener";
import { useDeskStore } from "@/stores/desk-store";
import { useInView } from "@/hooks/use-in-view";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { DeskBoardCard } from "@/lib/graphql/desk-board-card";
import { board as boardQueries } from "@/lib/queries/board";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation } from "@trenova/shared/types/permission";
import { useQueries, useQuery } from "@tanstack/react-query";
import {
  useLayoutEffect,
  useMemo,
  useState,
  type CSSProperties,
  type ReactElement,
  type ReactNode,
} from "react";
import { callLayoutFrom, isBoardExhibit, type BoardArtifact } from "../../artifact-payloads";
import { recordCardFor } from "../../cards/registry";
import { ArtIcon } from "../../desk-art-kinds";
import { cardTime } from "../../record-card/record-card-values";
import {
  boardArrival,
  CONVERSATION_ROWS,
  halfBlocks,
  readStamp,
  replyFootPlace,
  wordPieces,
  writtenSegments,
} from "./board-card";
import { Exhibit, Note, type OpenRecord, type OpensInPlace } from "./board-parts";
import { ExhibitSketch, WordsSketch, exhibitNumerals } from "./board-skeleton";
import { BoardActions } from "./board-work";
import { BoardWaitRow } from "./board-wait-row";
import { BoardWords, type CiteLight } from "./board-words";
import { BlockBody } from "./desk-board-body";

/** How far below the viewport a reply's board starts being read, so it is ready as it scrolls in. */
const READ_AHEAD = "600px 0px";

/**
 * What pulls a block with no title tab up to the block above it. The rows of
 * the desk stand a tab's height apart, which is room an exhibit's tab fills
 * and that words and buttons left empty.
 */
const NO_TAB = "-mt-7 first:mt-0";

/**
 * The exhibits under a reply's words: two half blocks side by side, each as tall
 * as its own content and aligned at the top, stacked when the reply is too
 * narrow for both to take a record row's narrow anatomy whole. That row needs
 * about 300px of exhibit ("S26110009560805" beside "252d 11h 5m behind" on one
 * line), so two sit side by side from 38rem: (608 - 12) / 2 = 298px each.
 */
function ReplyDesk({ children }: { children: ReactNode }) {
  return (
    <div className="@container mt-11">
      <div className="grid grid-cols-1 items-start gap-x-3 gap-y-11 @min-[38rem]:grid-cols-2">
        {children}
      </div>
    </div>
  );
}

/**
 * Where a block of words or buttons sits on the reply's desk, as the drawn
 * board sets it: a note in its column, words and buttons with no tab pulled up
 * to the block above.
 */
function wordsPlace(type: string, span: string | undefined, half: boolean): string {
  if (type === "note") return cn(span);
  if (type === "actions") return cn(!half && NO_TAB);
  return cn(span, !half && NO_TAB);
}

/**
 * A reply's board while it is read: the words as written, each reading a blank
 * where its value will stand, and every block in the shape and place it will
 * be drawn, with each list cut where the conversation cuts it, the count of
 * what it holds beyond, and the foot where the drawn board puts it
 * (`replyFootPlace`) where the surface opens boards. Under it, the row the reply's actions will take says the board
 * is being read. With no layout in view, the reply's own words stand in.
 */
function ReplyBoardWaiting({
  layout,
  fallback,
}: {
  layout: BoardArtifact;
  fallback: ReactElement | null;
}) {
  const t = useT();
  const opens = useArtifactOpener() !== null;
  const halves = halfBlocks(layout.blocks);
  const numerals = exhibitNumerals(layout.blocks);
  const last = layout.blocks.length - 1;
  const footPlace = replyFootPlace(layout.blocks, halves, isBoardExhibit);
  return (
    <div aria-busy="true" aria-label={t("Reading the board…")}>
      {layout.verdict !== "" ? (
        <div className="dk-prose">
          <p className="text-pretty">
            <BoardWords segments={writtenSegments(layout.verdict)} board={null} />
          </p>
        </div>
      ) : (
        fallback
      )}
      {layout.blocks.length > 0 && (
        <ReplyDesk>
          {layout.blocks.map((block, index) => {
            const span = halves[index] ? undefined : "col-span-full";
            if (!isBoardExhibit(block.type)) {
              return (
                <WordsSketch
                  // oxlint-disable-next-line react/no-array-index-key -- blocks keep their place
                  key={index}
                  block={block}
                  half={halves[index]}
                  className={wordsPlace(block.type, span, halves[index])}
                />
              );
            }
            const foot = index === last && opens && footPlace === "inside";
            const rowsFoot = foot && block.type === "records";
            return (
              <ExhibitSketch
                // oxlint-disable-next-line react/no-array-index-key -- blocks keep their place
                key={index}
                block={block}
                numeral={numerals[index]}
                half={halves[index]}
                className={span}
                cut={{
                  limit: CONVERSATION_ROWS,
                  after: (hidden) =>
                    hidden > 0 || rowsFoot ? (
                      <>
                        {hidden > 0 && <MoreRows hidden={hidden} onOpen={null} />}
                        {rowsFoot && <ReplyFoot onOpen={null} />}
                      </>
                    ) : null,
                }}
              >
                {foot && !rowsFoot && <ReplyFoot onOpen={null} className="-mx-4 mt-3.5 -mb-4" />}
              </ExhibitSketch>
            );
          })}
        </ReplyDesk>
      )}
      {footPlace === "under" && opens && <ReplyFoot onOpen={null} className="mt-4" />}
      <BoardWaitRow />
    </div>
  );
}

/**
 * A reply whose board could not be read: its own words, which say the answer
 * with the values written in, then a notice where the exhibits stand, with a
 * way to read them again.
 */
function ReplyBoardUnread({
  layout,
  fallback,
  answeredAt,
  retrying,
  onRetry,
}: {
  layout: BoardArtifact;
  fallback: ReactElement | null;
  answeredAt: number;
  retrying: boolean;
  onRetry: () => void;
}) {
  const t = useT();
  const first = layout.blocks.find((block) => isBoardExhibit(block.type));
  const notice = (
    <section
      className={cn("dk-bd-ex", first === undefined && "mt-3")}
      aria-label={t("The exhibits could not be read")}
    >
      {first && (
        <div className="dk-bd-tab">
          <span className="dk-bd-n">1</span>
          <span>{first.title || t("Exhibit {0}", 1)}</span>
        </div>
      )}
      <div className="flex items-center gap-3">
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <b className="text-dsk-base font-medium text-dsk-fg">
            {t("The exhibits could not be read just now")}
          </b>
          <span className="text-xs text-dsk-subtle">
            {fallback
              ? t("The words above keep what was read at {0}.", cardTime(answeredAt))
              : t("The board keeps what was read at {0}.", cardTime(answeredAt))}
          </span>
        </span>
        <Button size="sm" isLoading={retrying} loadingText={t("Reading again…")} onClick={onRetry}>
          {t("Try again")}
        </Button>
      </div>
    </section>
  );
  return (
    <>
      {fallback}
      {first ? <div className="mt-11">{notice}</div> : notice}
    </>
  );
}

/** The arrival classes and timings of one block of an arriving board, or none. */
function arriving(
  arrival: ReturnType<typeof boardArrival> | null,
  index: number,
): { className?: string; style?: CSSProperties } {
  if (!arrival) return {};
  return {
    className: "dk-bd-arrive",
    style: {
      "--dk-bd-at": `${arrival.exhibitAt(index)}ms`,
      "--dk-bd-rows-at": `${arrival.rowsAt(index)}ms`,
    } as CSSProperties,
  };
}

/**
 * The foot of a reply's board: where it came from, and the way to open it
 * beside the conversation. While the board is read (`onOpen` null) the same box
 * holds a blank where the button will be.
 */
function ReplyFoot({ onOpen, className }: { onOpen: (() => void) | null; className?: string }) {
  const t = useT();
  return (
    <div
      className={cn(
        "flex min-h-11 items-center justify-between gap-3 border-t border-dsk-b-sub py-2 pr-3 pl-4 text-xs text-dsk-subtle",
        className,
      )}
    >
      <span>{t("From the board for this answer")}</span>
      {onOpen ? (
        <Button variant="secondary" size="sm" onClick={onOpen}>
          {t("Open the board")}
          <ArtIcon name="right" size={13} />
        </Button>
      ) : (
        <i className="block h-7 w-32 shrink-0 ui-shimmer rounded-md" aria-hidden="true" />
      )}
    </div>
  );
}

/** How many more of a list the board holds than the conversation draws, opening the board. */
function MoreRows({ hidden, onOpen }: { hidden: number; onOpen: (() => void) | null }) {
  const t = useT();
  const label = t("{0, plural, one {# more on the board} other {# more on the board}}", hidden);
  const className =
    "flex min-h-10 w-full items-center border-t border-dsk-b-sub px-4 text-left text-dsk-base text-dsk-subtle";
  return onOpen ? (
    <button
      type="button"
      className={cn(className, "cursor-pointer ui-focus-ring hover:bg-dsk-hover hover:text-dsk-fg")}
      onClick={onOpen}
    >
      {label}
    </button>
  ) : (
    <span className={className}>{label}</span>
  );
}

/**
 * A board drawn as the reply: its verdict in the conversation's own words,
 * readings in it, cited phrases that light the exhibit they cite, then each
 * block as an exhibit cut to the conversation's width, lists cut to six rows,
 * and a foot that opens the board beside the conversation. A record opens
 * beside the conversation as its own card, in the board's place, with the way
 * back to the board; the board itself opens only from its foot, or from a list
 * that holds more than the conversation draws.
 */
export function ReplyBoardView({ card, arrives }: { card: DeskBoardCard; arrives: boolean }) {
  const t = useT();
  const now = useNowSeconds();
  const openArtifact = useArtifactOpener();
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const [lit, setLit] = useState(0);
  const halves = halfBlocks(card.blocks);
  const stamp = readStamp(card, t);
  const openRecord = useDeskStore((state) => state.openArtifactRecord);
  const openBoard = openArtifact
    ? () => {
        openRecord(null);
        openArtifact(card.artifactId);
      }
    : null;
  const opensInPlace: OpensInPlace = (resource) => {
    const entry = recordCardFor(resource);
    return openBoard !== null && entry !== null && hasPermission(entry.resource, Operation.Read);
  };
  const open: OpenRecord = (row, order) => {
    if (!openArtifact) return;
    openRecord({ artifactId: card.artifactId, entity: row.resource, recordId: row.id, order });
    openArtifact(card.artifactId);
  };
  const cite: CiteLight = { lit, onLight: setLit };
  const arrival = arrives
    ? boardArrival(wordPieces(card.verdict).phrases, card.blocks.length)
    : null;
  const last = card.blocks.length - 1;
  const footPlace = replyFootPlace(card.blocks, halves, isBoardExhibit);

  return (
    <article aria-label={card.title}>
      {card.verdict.length > 0 && (
        <div className="dk-prose">
          <p className="text-pretty">
            <BoardWords
              segments={card.verdict}
              board={card}
              cite={cite}
              arrival={arrival ? { landAt: arrival.landAt } : undefined}
            />
          </p>
        </div>
      )}
      {card.blocks.length > 0 && (
        <ReplyDesk>
          {card.blocks.map((block, index) => {
            const key = `${block.type}-${index}`;
            const span = halves[index] ? undefined : "col-span-full";
            const motion = arriving(arrival, index);
            if (block.type === "prose") {
              return (
                <p
                  key={key}
                  className={cn("dk-bd-prose", span, !halves[index] && NO_TAB, motion.className)}
                  style={motion.style}
                >
                  <BoardWords segments={block.text} board={card} cite={cite} />
                </p>
              );
            }
            if (block.type === "actions") {
              return (
                <BoardActions
                  key={key}
                  actions={block.actions}
                  threadId={card.threadId}
                  half={halves[index]}
                  className={cn(!halves[index] && NO_TAB, motion.className)}
                  style={motion.style}
                />
              );
            }
            if (block.type === "note") {
              return (
                <Note
                  key={key}
                  block={block}
                  board={card}
                  half={halves[index]}
                  open={open}
                  opensInPlace={opensInPlace}
                  className={cn(span, motion.className)}
                  style={motion.style}
                />
              );
            }
            const foot = index === last && openBoard !== null && footPlace === "inside";
            const rowsFoot = foot && block.type === "records" && block.state === "ready";
            return (
              <Exhibit
                key={key}
                block={block}
                board={card}
                stamp={stamp}
                half={halves[index]}
                lit={block.numeral > 0 && lit === block.numeral}
                className={cn(span, motion.className)}
                style={motion.style}
              >
                <BlockBody
                  block={block}
                  card={card}
                  now={now}
                  open={open}
                  opensInPlace={opensInPlace}
                  cut={{
                    limit: CONVERSATION_ROWS,
                    after: (hidden) =>
                      hidden > 0 || rowsFoot ? (
                        <>
                          {hidden > 0 && <MoreRows hidden={hidden} onOpen={openBoard} />}
                          {rowsFoot && openBoard && <ReplyFoot onOpen={openBoard} />}
                        </>
                      ) : null,
                  }}
                />
                {foot && !rowsFoot && openBoard && (
                  <ReplyFoot onOpen={openBoard} className="-mx-4 mt-3.5 -mb-4" />
                )}
              </Exhibit>
            );
          })}
        </ReplyDesk>
      )}
      {footPlace === "under" && openBoard && <ReplyFoot onOpen={openBoard} className="mt-4" />}
    </article>
  );
}

function layoutOf(item: ReplyBoard): BoardArtifact {
  return callLayoutFrom(item.tool, item.spec ?? {}, {
    title: item.artifact.title,
    author: "",
    place: "conversation",
  });
}

/**
 * One board a reply draws. Nothing is read until the reply comes within reach
 * of the viewport, so a long conversation reads only what is on screen; the
 * read is the pane's own (`board.deskCard`), so opening the board beside the
 * conversation reads nothing again.
 */
function ReplyBoardItem({
  item,
  fallback,
  answeredAt,
  arrives,
}: {
  item: ReplyBoard;
  fallback: ReactElement | null;
  answeredAt: number;
  arrives: boolean;
}) {
  const layout = useMemo(() => layoutOf(item), [item]);
  const [probe, near] = useInView<HTMLDivElement>({ rootMargin: READ_AHEAD, once: true });
  const query = useQuery({
    ...boardQueries.deskCard(item.artifact.threadId, item.artifact.id),
    enabled: near,
    retry: false,
  });

  return (
    <div ref={probe}>
      {query.data ? (
        <ReplyBoardView card={query.data} arrives={arrives} />
      ) : query.isError ? (
        <ReplyBoardUnread
          layout={layout}
          fallback={fallback}
          answeredAt={answeredAt}
          retrying={query.isFetching}
          onRetry={() => void query.refetch()}
        />
      ) : (
        // The reply's words stand in only when the call's layout is not in view;
        // with it, the skeleton is the one the running turn drew, so nothing
        // changes when the reply is saved.
        <ReplyBoardWaiting layout={layout} fallback={item.spec === null ? fallback : null} />
      )}
    </div>
  );
}

/**
 * The boards a reply's turn placed in the conversation, drawn as the reply. A
 * reply that landed while the person watched arrives: its words phrase by
 * phrase, its exhibits rising, its rows following and its readings landing.
 * One read from history is drawn at once.
 */
export function DeskReplyBoards({
  boards,
  fallback,
  answeredAt,
  arrives,
  onDrawn,
}: ReplyBoardsProps) {
  // Watches each board's read without starting it: the items read when they
  // come into reach.
  const reads = useQueries({
    queries: boards.map((item) => ({
      ...boardQueries.deskCard(item.artifact.threadId, item.artifact.id),
      enabled: false,
    })),
  });
  const drawn = reads.every((read) => read.data !== undefined || read.isError);
  useLayoutEffect(() => {
    if (drawn) onDrawn?.();
  }, [drawn, onDrawn]);

  // `arrives` is settled by the reply when it mounts (DeskReply), so the moment
  // this code loads never decides it.
  return boards.map((item, index) => (
    <ReplyBoardItem
      key={item.artifact.id}
      item={item}
      fallback={index === 0 ? fallback : null}
      answeredAt={answeredAt}
      arrives={arrives}
    />
  ));
}

/** A board the running turn is still showing, drawn as its skeleton until the reply is saved. */
export function DeskPendingBoard({ tool, spec }: PendingBoardProps) {
  const layout = useMemo(
    () => callLayoutFrom(tool, spec ?? {}, { title: "", author: "", place: "conversation" }),
    [spec, tool],
  );
  return <ReplyBoardWaiting layout={layout} fallback={null} />;
}
