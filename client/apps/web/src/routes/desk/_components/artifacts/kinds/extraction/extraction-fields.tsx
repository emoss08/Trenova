import { EyeOffIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, type KeyboardEvent } from "react";
import { GROUP_LABELS, rowsInOrder, type FieldGroup, type FieldRow } from "./extraction-groups";
import {
  rowAnnouncement,
  rowDomId,
  rowReason,
  sharedStopReason,
  stopTitle,
} from "./extraction-words";

type ListProps = {
  listId: string;
  groups: readonly FieldGroup[];
  multiPage: boolean;
  page: number;
  selected: string | null;
  hot: string | null;
  onSelect: (key: string, from: "pointer" | "keyboard") => void;
  onClear: () => void;
  onPoint: (key: string | null) => void;
};

/**
 * What was read, grouped the way a dispatcher reads a load, as one list box: the arrow keys,
 * Home and End walk it and each step lights the field's box on the page, Escape lets the
 * selection go. A settled value is plain; one read short of sure shows its figure; one that
 * needs a person says why under it; one the document lacked says so where it would have been.
 */
export function ExtractionFields({
  listId,
  groups,
  multiPage,
  page,
  selected,
  hot,
  onSelect,
  onClear,
  onPoint,
}: ListProps) {
  const t = useT();
  const listRef = useRef<HTMLDivElement>(null);
  const rows = rowsInOrder(groups);

  useEffect(() => {
    if (selected === null) return;
    listRef.current?.ownerDocument
      .getElementById(rowDomId(listId, selected))
      ?.scrollIntoView?.({ block: "nearest" });
  }, [listId, selected]);

  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.altKey || event.ctrlKey || event.metaKey || rows.length === 0) return;
    const at = rows.findIndex((row) => row.key === selected);
    const moves: Record<string, () => number> = {
      ArrowDown: () => (at < 0 ? 0 : Math.min(rows.length - 1, at + 1)),
      ArrowUp: () => (at < 0 ? 0 : Math.max(0, at - 1)),
      Home: () => 0,
      End: () => rows.length - 1,
    };
    if (event.key === "Escape" && selected !== null) {
      event.preventDefault();
      onClear();
      return;
    }
    const move = moves[event.key];
    if (!move) return;
    event.preventDefault();
    onSelect(rows[move()].key, "keyboard");
  };

  const renderRow = (row: FieldRow, shared: string | null = null) => {
    const own = rowReason(row, t);
    const reason = own !== null && own === shared ? null : own;
    const field = row.kind === "read" ? row.field : null;
    const elsewhere = field !== null && multiPage && field.page > 0 && field.page !== page;
    const unplaced = field !== null && field.box === null;
    return (
      // oxlint-disable-next-line jsx-a11y/click-events-have-key-events -- the list box takes the keys for every option, as a list box does
      <div
        key={row.key}
        id={rowDomId(listId, row.key)}
        role="option"
        aria-selected={selected === row.key}
        aria-label={rowAnnouncement(row, multiPage, t)}
        className={cn("dk-ex-row", `dk-${row.state}`, hot === row.key && "dk-hot")}
        onClick={() => onSelect(row.key, "pointer")}
        onMouseEnter={() => onPoint(row.key)}
        onMouseLeave={() => onPoint(null)}
      >
        <span className="dk-ex-l">{row.label}</span>
        <span className="dk-ex-v">{field ? field.value : t("Not found")}</span>
        <span className="dk-ex-m">
          {row.state === "fair" && field && (
            <span>{t("{0}%", Math.round(field.confidence * 100))}</span>
          )}
          {elsewhere && <span>{t("p. {0}", field.page)}</span>}
          {unplaced && (
            <span title={t("Read, but not found on the page")}>
              <EyeOffIcon className="size-3" />
            </span>
          )}
        </span>
        {reason !== null && (
          <span className="dk-ex-why">
            <i aria-hidden className={row.kind === "missing" ? "dk-ex-ring" : "dk-ex-dot"} />
            {reason}
            {field && field.reason === "conflict" && field.readings.length > 1 && (
              <span className="dk-ex-alts">
                {field.readings.map((reading) => (
                  <code key={reading} className="dk-ex-alt">
                    {reading}
                  </code>
                ))}
              </span>
            )}
          </span>
        )}
      </div>
    );
  };

  return (
    // oxlint-disable-next-line jsx-a11y/no-noninteractive-tabindex -- a list box is focused as one control and its options are walked with the arrow keys
    <div
      ref={listRef}
      className="dk-ex-fields ui-focus-ring"
      role="listbox"
      tabIndex={0}
      aria-label={t("What was read")}
      aria-activedescendant={selected !== null ? rowDomId(listId, selected) : undefined}
      onKeyDown={onKeyDown}
    >
      {groups.map((group) => {
        const headId = `${listId}-${group.id}`;
        const checks =
          group.stops.filter((stop) => stop.doubtful).length +
          group.rows.filter((row) => row.state === "doubtful" || row.state === "missing").length;
        return (
          <div key={group.id} role="group" aria-labelledby={headId} className="dk-ex-grp">
            <div className="dk-ex-grp-h" aria-hidden>
              <span id={headId}>{GROUP_LABELS[group.id]}</span>
              {checks > 0 && <span>{t("{0} to look at", checks)}</span>}
            </div>
            {group.stops.map((stop) => {
              const stopId = `${headId}-${stop.sequence}`;
              const shared = sharedStopReason(stop, t);
              return (
                <div
                  key={stop.sequence}
                  role="group"
                  aria-labelledby={stopId}
                  className={cn("dk-ex-stop", stop.doubtful && "dk-doubtful")}
                >
                  <div className="dk-ex-stop-h">
                    <span className="dk-ex-stop-n" aria-hidden>
                      {stop.sequence}
                    </span>
                    <span id={stopId}>{stopTitle(stop, t)}</span>
                    {shared !== null && (
                      <span className="dk-ex-stop-why">
                        <i aria-hidden className="dk-ex-dot" />
                        {shared}
                      </span>
                    )}
                    {stop.rows.some(
                      (row) => row.kind === "read" && row.field.stop?.appointment,
                    ) && <span className="dk-ex-appt">{t("Appointment")}</span>}
                  </div>
                  {stop.rows.map((row) => renderRow(row, shared))}
                </div>
              );
            })}
            {group.rows.map((row) => renderRow(row))}
          </div>
        );
      })}
    </div>
  );
}
