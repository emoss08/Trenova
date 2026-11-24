import { cn } from "@trenova/shared/lib/utils";
import { readableIds, toneClass } from "../board-card";
import { RecordChip, type OpenRecord, type OpensInPlace } from "../board-parts";
import { cardMomentAfter } from "../../../record-card/record-card-values";
import type { DeskBoardLogEntry } from "./metric-types";

/**
 * A log, entry by entry: when, what became of it (its tone on the dot), and under it the detail
 * with the records it names, each opening its record. Every entry is two fixed lines, the text
 * giving way with an ellipsis and its whole in `title`, so the log is the height its sketch
 * reserves.
 */
export function BoardLog({
  entries,
  now,
  open,
  opensInPlace,
}: {
  entries: DeskBoardLogEntry[];
  now: number;
  open: OpenRecord;
  opensInPlace: OpensInPlace;
}) {
  return (
    <ol className="dk-bd-log dk-bd-mlog">
      {entries.map((entry, index) => {
        const order = readableIds(entry.records);
        const named = entry.records.filter((row) => row.readable);
        return (
          // oxlint-disable-next-line react/no-array-index-key -- the log is a fixed run of entries
          <li key={index} className={cn(toneClass(entry.tone))}>
            <time dateTime={new Date(entry.at * 1000).toISOString()}>
              {cardMomentAfter(now, entry.at)}
            </time>
            <i />
            <div>
              <span title={entry.text}>{entry.text}</span>
              {(entry.detail !== "" || named.length > 0) && (
                <em title={entry.detail || undefined}>
                  {entry.detail}
                  {named.map((row) => (
                    <RecordChip
                      key={row.id}
                      row={row}
                      order={order}
                      open={open}
                      opensInPlace={opensInPlace}
                    />
                  ))}
                </em>
              )}
            </div>
          </li>
        );
      })}
    </ol>
  );
}
