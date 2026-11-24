import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { describeApiError } from "@/lib/api-error-message";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { DeskBoardBlock, DeskBoardCard, DeskBoardRow } from "@/lib/graphql/desk-board-card";
import { board } from "@/lib/queries/board";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useMemo, type ReactNode } from "react";
import { boardFrom } from "../../artifact-payloads";
import { InPlaceRecord, recordCardFor, usePreloadRecordCard } from "../../cards/registry";
import { ArtIcon } from "../../desk-art-kinds";
import { cardTime } from "../../record-card/record-card-values";
import { Notice } from "../artifact-body-parts";
import {
  boardReads,
  cutRows,
  halfBlocks,
  initialsOf,
  opened as openedRecord,
  readableIds,
  readStamp,
} from "./board-card";
import { BoardMetric, NothingToDraw } from "./blocks/board-metric";
import {
  BoardTable,
  Checks,
  Empty,
  Exhibit,
  HeadToHead,
  LogBook,
  Note,
  Pips,
  Readout,
  RecordRows,
  Unreadable,
  type OpenRecord,
  type OpensInPlace,
} from "./board-parts";
import { BoardSkeleton } from "./board-skeleton";
import { BoardActions, BoardWork } from "./board-work";
import { BoardWords, Numeral } from "./board-words";

/**
 * The records a board names, opened in its place as each record's own card, stepping through
 * the block that listed them; a kind without a card here, or one the reader may not read,
 * opens on its page instead. Which record is open is the Desk's to know, not the board's: a
 * row of this board clicked in the conversation opens the pane at that record.
 */
function useBoardRecords(
  artifactId: string,
  backLabel: string,
): {
  open: OpenRecord;
  opensInPlace: OpensInPlace;
  opened: ReactNode;
} {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const opened = useDeskStore((state) =>
    state.artifactRecord?.artifactId === artifactId ? state.artifactRecord : null,
  );
  const setOpened = useDeskStore((state) => state.openArtifactRecord);
  const entry = opened ? recordCardFor(opened.entity) : null;
  usePreloadRecordCard(entry);

  return {
    opensInPlace: (kind) => {
      const card = recordCardFor(kind);
      return card !== null && hasPermission(card.resource, Operation.Read);
    },
    open: (row: DeskBoardRow, order) => {
      const target = openedRecord(row);
      setOpened({ artifactId, entity: target.kind, recordId: target.id, order });
    },
    opened:
      opened && entry ? (
        <InPlaceRecord
          entry={entry}
          recordId={opened.recordId}
          order={opened.order}
          onSelect={(recordId) => setOpened({ ...opened, recordId })}
          onBack={() => setOpened(null)}
          backLabel={backLabel}
        />
      ) : null,
  };
}

/** A board the agent laid out, read now: words first, each block filled from what it names. */
export function DeskBoardBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const layout = useMemo(() => boardFrom(artifact), [artifact]);
  const query = useQuery({ ...board.deskCard(artifact.threadId, artifact.id), retry: false });
  const records = useBoardRecords(artifact.id, t("Back to the board"));

  if (records.opened) {
    return records.opened;
  }
  if (query.isPending) {
    return <BoardSkeleton layout={layout} />;
  }
  if (query.isError) {
    return (
      <Notice>
        {describeApiError(
          query.error,
          t("This board could not be read. It may belong to someone else."),
        )}
      </Notice>
    );
  }

  return <BoardView card={query.data} open={records.open} opensInPlace={records.opensInPlace} />;
}

/**
 * Where a list is cut, and what stands after its rows: the conversation draws
 * six, then says how many more the board holds (`hidden`, 0 when none were
 * cut) and, under the last exhibit, the way to the board.
 */
export type RowCut = { limit: number; after: (hidden: number) => ReactNode };

/** What an exhibit holds, drawn from its filled block. */
export function BlockBody({
  block,
  card,
  now,
  open,
  opensInPlace,
  cut,
}: {
  block: DeskBoardBlock;
  card: DeskBoardCard;
  now: number;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
  cut?: RowCut;
}) {
  if (block.state === "unreadable") return <Unreadable block={block} />;
  if (block.state === "empty") {
    return block.metric ? <NothingToDraw form={block.metric.form} /> : <Empty block={block} />;
  }
  switch (block.type) {
    case "figures":
      return <Readout figures={block.figures} answeredAt={card.answeredAt} now={now} />;
    case "records": {
      const rows = cut ? cutRows(block.rows, cut.limit) : { shown: block.rows, more: 0 };
      return (
        <RecordRows
          rows={rows.shown}
          order={readableIds(block.rows)}
          now={now}
          open={open}
          opensInPlace={opensInPlace}
          more={cut?.after(rows.more)}
        />
      );
    }
    case "compare":
      return block.compare ? (
        <HeadToHead compare={block.compare} open={open} opensInPlace={opensInPlace} />
      ) : null;
    case "table":
      return block.table ? <BoardTable table={block.table.table} /> : null;
    case "chart":
    case "metric":
      return block.metric ? (
        <BoardMetric
          metric={block.metric}
          answeredAt={card.answeredAt}
          now={now}
          open={open}
          opensInPlace={opensInPlace}
        />
      ) : null;
    case "timeline":
      return <LogBook events={block.events} now={now} />;
    case "checks":
      return <Checks checks={block.checks} open={open} opensInPlace={opensInPlace} />;
    case "meter":
      return block.meter ? <Pips meter={block.meter} /> : null;
    case "work":
      return <BoardWork rows={block.work} now={now} />;
    default:
      return null;
  }
}

/** A board as the reader sees it now. */
export function BoardView({
  card,
  open,
  opensInPlace,
}: {
  card: DeskBoardCard;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  const t = useT();
  const now = useNowSeconds();
  const { copy, text: copiedText } = useCopyToClipboard();
  const halves = halfBlocks(card.blocks);
  const reads = boardReads(card, t);
  const read = cardTime(card.readAt);
  const stamp = readStamp(card, t);

  return (
    <article className="dk-bd" aria-label={card.title}>
      <header className="dk-bd-mast">
        <div className="dk-bd-mast-top">
          <span>{t("Board")}</span>
          <i />
          <span>{t("{0, plural, one {# exhibit} other {# exhibits}}", reads.length)}</span>
          <span className="dk-bd-live">
            <i />
            {t("Live · read {0}", read)}
          </span>
        </div>
        <h2 className="dk-bd-title">{card.title}</h2>
        {card.verdict.length > 0 && (
          <p className="dk-bd-verdict">
            <BoardWords segments={card.verdict} board={card} />
          </p>
        )}
        {card.author !== "" && (
          <div className="dk-bd-by">
            <span className="dk-bd-agent" aria-hidden="true">
              {initialsOf(card.author)}
            </span>
            <span>{t("{0} answered at {1}", card.author, cardTime(card.answeredAt))}</span>
          </div>
        )}
      </header>
      <div className="dk-bd-desk">
        {card.blocks.map((block, index) => {
          const key = `${block.type}-${index}`;
          if (block.type === "prose") {
            return (
              <p key={key} className={cn("dk-bd-prose", halves[index] && "dk-bd-half")}>
                <BoardWords segments={block.text} board={card} />
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
              />
            );
          }
          return (
            <Exhibit key={key} block={block} board={card} half={halves[index]} stamp={stamp}>
              <BlockBody
                block={block}
                card={card}
                now={now}
                open={open}
                opensInPlace={opensInPlace}
              />
            </Exhibit>
          );
        })}
      </div>
      <footer className="dk-bd-foot">
        <span>
          {t("Reads")}
          {reads.map((entry) => (
            <span key={entry.numeral} className="dk-bd-foot-read">
              <Numeral n={entry.numeral} struck={entry.struck} />
              {entry.phrase}
            </span>
          ))}
        </span>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void copy(card.markdown, { withToast: false })}
        >
          <ArtIcon name={copiedText === card.markdown ? "check" : "copy"} size={13} />
          {copiedText === card.markdown ? t("Copied") : t("Copy as text")}
        </Button>
      </footer>
    </article>
  );
}
