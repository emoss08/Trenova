import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CardIcon, CardSection } from "../../record-card/record-card";
import type { CheckItem, FieldRow, ReviewTally } from "./extraction-groups";
import type { PageSize } from "./extraction-payload";
import { ExtractionSnippet } from "./extraction-snippet";
import { checkFields } from "./extraction-summary";
import { rowReason, sharedStopReason, stopTitle } from "./extraction-words";

type ReadRow = FieldRow & { kind: "read" };

type CheckProps = {
  documentId: string;
  pages: readonly PageSize[];
  tally: ReviewTally;
  selected: string | null;
  onSelect: (key: string) => void;
  /** Finds a value on the page: selects it and brings the document into view. */
  onShow: (key: string) => void;
};

/** The rows of an item that need a person: the field itself, or a stop's doubtful parts. */
function itemRows(item: CheckItem): ReadRow[] {
  if (item.kind === "field") return [item.row];
  return item.stop.rows.filter(
    (row): row is ReadRow => row.kind === "read" && row.state === "doubtful",
  );
}

/**
 * What a person settles before the reading becomes a shipment: each doubtful value with why,
 * beside the bit of page it came from, a conflict's readings side by side, then each field the
 * reading looked for and did not find. Nothing here is red: a value to check is amber, a value
 * not read is a quiet dashed ring, because neither is yet a mistake.
 */
export function ExtractionCheck({
  documentId,
  pages,
  tally,
  selected,
  onSelect,
  onShow,
}: CheckProps) {
  const t = useT();
  if (tally.check.length === 0 && tally.missing.length === 0) return null;

  return (
    <CardSection
      title={t("Check before it becomes a shipment")}
      aside={
        <span>
          {[
            tally.check.length > 0
              ? t("{0, plural, one {# to check} other {# to check}}", tally.check.length)
              : "",
            tally.missing.length > 0
              ? t("{0, plural, one {# not read} other {# not read}}", tally.missing.length)
              : "",
          ]
            .filter(Boolean)
            .join(" · ")}
        </span>
      }
    >
      <ol className="dk-ex-check">
        {tally.check.map((item) => {
          const rows = itemRows(item);
          const first = rows[0];
          if (!first) return null;
          const title = item.kind === "stop" ? stopTitle(item.stop, t) : first.label;
          const shared = item.kind === "stop" ? sharedStopReason(item.stop, t) : null;
          const placed = checkFields(rows).filter((field) => field.box !== null);
          const page = placed[0]?.page ?? 0;
          const boxes = placed.flatMap((field) =>
            field.page === page && field.box ? [field.box] : [],
          );
          const on = rows.some((row) => row.key === selected);
          return (
            <li
              key={item.kind === "stop" ? `stop:${item.stop.sequence}` : first.key}
              className={cn("dk-ex-ck", on && "dk-on")}
            >
              <span className="dk-ex-ck-mark" aria-hidden>
                <CardIcon name="warn" size={12} stroke={2.2} />
              </span>
              <Button
                variant="bare"
                size="bare"
                className="dk-ex-ck-main ui-focus-ring"
                aria-pressed={on}
                onClick={() => onSelect(first.key)}
              >
                <span className="dk-ex-ck-l">{title}</span>
                {rows.map((row) => (
                  <CheckLine key={row.key} row={row} part={item.kind === "stop"} shared={shared} />
                ))}
                {shared !== null && <span className="dk-ex-ck-why">{shared}</span>}
              </Button>
              {page > 0 && documentId !== "" && (
                <ExtractionSnippet
                  documentId={documentId}
                  page={page}
                  pages={pages}
                  boxes={boxes}
                  label={title}
                  doubtful
                  onShow={() => onShow(first.key)}
                />
              )}
            </li>
          );
        })}
        {tally.missing.map((row) => (
          <li key={row.key} className="dk-ex-ck dk-miss">
            <span className="dk-ex-ck-mark" aria-hidden />
            <span className="dk-ex-ck-main">
              <span className="dk-ex-ck-l">{row.label}</span>
              <span className="dk-ex-ck-v dk-blank">{t("Not read")}</span>
              <span className="dk-ex-ck-why">{t("Not found on any page that was read")}</span>
            </span>
          </li>
        ))}
      </ol>
    </CardSection>
  );
}

/** One doubtful value: the part of the stop it is, the value, and why it needs a look. */
function CheckLine({ row, part, shared }: { row: ReadRow; part: boolean; shared: string | null }) {
  const t = useT();
  const reason = rowReason(row, t);
  const { field } = row;
  const conflict = field.reason === "conflict" && field.readings.length > 1;
  return (
    <span className="dk-ex-ck-line">
      {conflict ? (
        <span className="dk-ex-pick">
          {part && <span className="dk-ex-ck-part">{row.label}</span>}
          {field.readings.map((reading) => (
            <span key={reading} className={cn(reading === field.value && "dk-used")}>
              <b>{reading}</b>
              {reading === field.value && <em>{t("used")}</em>}
            </span>
          ))}
        </span>
      ) : (
        <span className="dk-ex-ck-v">
          {part && <span className="dk-ex-ck-part">{row.label}</span>}
          {field.value}
        </span>
      )}
      {reason !== null && reason !== shared && <span className="dk-ex-ck-why">{reason}</span>}
    </span>
  );
}
