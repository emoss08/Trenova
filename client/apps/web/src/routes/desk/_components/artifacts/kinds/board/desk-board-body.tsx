import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { describeApiError } from "@/lib/api-error-message";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { DeskBoardBlock, DeskBoardCard, DeskBoardRow } from "@/lib/graphql/desk-board-card";
import { board } from "@/lib/queries/board";
import type { AssistantArtifact } from "@/types/assistant";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState, type ReactNode } from "react";
import { boardFrom } from "../../artifact-payloads";
import { InPlaceRecord, recordCardFor, usePreloadRecordCard } from "../../cards/registry";
import { ArtIcon } from "../../desk-art-kinds";
import { cardTime } from "../../record-card/record-card-values";
import { Notice } from "../artifact-body-parts";
import { boardReads, halfBlocks, initialsOf, readStamp } from "./board-card";
import {
  BoardChart,
  BoardTable,
  Checks,
  Empty,
  Exhibit,
  HeadToHead,
  LogBook,
  Note,
  Numeral,
  Pips,
  Readout,
  RecordRows,
  Segments,
  Unreadable,
  type OpenRecord,
  type OpensInPlace,
} from "./board-parts";
import { BoardSkeleton } from "./board-skeleton";

type Opened = { entity: string; recordId: string; order: readonly string[] };

/**
 * The records a board names, opened in its place as each record's own card, stepping through
 * the block that listed them; a kind without a card here, or one the reader may not read,
 * opens on its page instead.
 */
function useBoardRecords(backLabel: string): {
  open: OpenRecord;
  opensInPlace: OpensInPlace;
  opened: ReactNode;
} {
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  const [opened, setOpened] = useState<Opened | null>(null);
  const entry = opened ? recordCardFor(opened.entity) : null;
  usePreloadRecordCard(entry);

  return {
    opensInPlace: (resource) => {
      const card = recordCardFor(resource);
      return card !== null && hasPermission(card.resource, Operation.Read);
    },
    open: (row: DeskBoardRow, order) =>
      setOpened({ entity: row.resource, recordId: row.id, order }),
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
  const records = useBoardRecords(t("Back to the board"));

  if (records.opened) {
    return records.opened;
  }
  if (query.isPending) {
    return <BoardSkeleton layout={layout} question={artifact.turn} />;
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

  return (
    <BoardView
      card={query.data}
      question={query.data.question || artifact.turn}
      open={records.open}
      opensInPlace={records.opensInPlace}
    />
  );
}

function BlockBody({
  block,
  card,
  now,
  open,
  opensInPlace,
}: {
  block: DeskBoardBlock;
  card: DeskBoardCard;
  now: number;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  if (block.state === "unreadable") return <Unreadable block={block} />;
  if (block.state === "empty") return <Empty block={block} />;
  switch (block.type) {
    case "figures":
      return <Readout figures={block.figures} answeredAt={card.answeredAt} now={now} />;
    case "records":
      return <RecordRows rows={block.rows} now={now} open={open} opensInPlace={opensInPlace} />;
    case "compare":
      return block.compare ? (
        <HeadToHead compare={block.compare} open={open} opensInPlace={opensInPlace} />
      ) : null;
    case "table":
      return block.table ? <BoardTable table={block.table.table} /> : null;
    case "chart":
      return block.chart ? <BoardChart chart={block.chart} /> : null;
    case "timeline":
      return <LogBook events={block.events} now={now} />;
    case "checks":
      return <Checks checks={block.checks} open={open} opensInPlace={opensInPlace} />;
    case "meter":
      return block.meter ? <Pips meter={block.meter} /> : null;
    default:
      return null;
  }
}

/** A board as the reader sees it now. */
export function BoardView({
  card,
  question,
  open,
  opensInPlace,
}: {
  card: DeskBoardCard;
  question: string;
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
        {question !== "" && (
          <div className="dk-bd-asked">
            <span>{t("You asked")}</span>
            <span className="dk-bd-bubble">{question}</span>
          </div>
        )}
        <h2 className="dk-bd-title">{card.title}</h2>
        {card.verdict.length > 0 && (
          <p className="dk-bd-verdict">
            <Segments segments={card.verdict} />
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
                <Segments segments={block.text} />
              </p>
            );
          }
          if (block.type === "note") {
            return (
              <Note
                key={key}
                block={block}
                half={halves[index]}
                open={open}
                opensInPlace={opensInPlace}
              />
            );
          }
          return (
            <Exhibit key={key} block={block} half={halves[index]} stamp={stamp}>
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
