import { CalendarIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { CardSection } from "../../record-card/record-card";
import { GROUP_LABELS, type FieldGroup, type FieldRow, type StopGroup } from "./extraction-groups";
import { readMoney } from "./extraction-summary";
import { rowAnnouncement } from "./extraction-words";

type ValuePick = {
  selected: string | null;
  hot: string | null;
  multiPage: boolean;
  onSelect: (key: string) => void;
  onPoint: (key: string | null) => void;
};

type LoadProps = ValuePick & { groups: readonly FieldGroup[] };

/** Keys whose values are money, set in figures and to the right. */
const MONEY_KEYS: ReadonlySet<string> = new Set(["rate", "fuelSurcharge"]);

/**
 * What the shipment would be, as the reading has it: the stops down a numbered route, then
 * who is involved, the freight, the references, the charges and anything else read, each in
 * a short column. Every value is a button that finds it on the page; a value read loosely is
 * underlined dotted, one that needs a person wavy amber, and one not read says so where it
 * would have been.
 */
export function ExtractionLoad({ groups, ...pick }: LoadProps) {
  const t = useT();
  const stops = groups.find((group) => group.id === "stops");
  const panels = groups.filter((group) => group.id !== "stops");

  return (
    <CardSection
      title={t("The load as read")}
      aside={<span>{t("what the shipment would be")}</span>}
    >
      {stops && stops.stops.length > 0 && (
        <ol className="dk-ex-route" aria-label={t("Stops")}>
          {stops.stops.map((stop) => (
            <RouteStop key={stop.sequence} stop={stop} {...pick} />
          ))}
        </ol>
      )}
      {(panels.length > 0 || (stops && stops.rows.length > 0)) && (
        <div className="dk-ex-facts">
          {stops && stops.rows.length > 0 && (
            <FactPanel title={t("Route")} rows={stops.rows} {...pick} />
          )}
          {panels.map((group) => (
            <FactPanel key={group.id} title={GROUP_LABELS[group.id]} rows={group.rows} {...pick} />
          ))}
        </div>
      )}
    </CardSection>
  );
}

function roleLabel(role: string, t: ReturnType<typeof useT>): string {
  switch (role) {
    case "pickup":
      return t("Pickup");
    case "delivery":
      return t("Delivery");
    default:
      return t("Stop");
  }
}

/** One stop on the route: its place in it, its role, and its name, address and window. */
function RouteStop({ stop, ...pick }: ValuePick & { stop: StopGroup }) {
  const t = useT();
  const parts = stop.rows.filter((row) => row.kind === "read" && row.field.stop !== null);
  const extras = stop.rows.filter((row) => !parts.includes(row));
  const appointment = parts.some((row) => row.kind === "read" && row.field.stop?.appointment);
  return (
    <li className={cn(stop.doubtful && "dk-warn")}>
      <span className="dk-ex-pin" aria-hidden>
        {stop.sequence}
      </span>
      <div>
        <span className="dk-ex-role">
          {roleLabel(stop.role, t)}
          {appointment && (
            <span className="dk-ex-tag">
              <CalendarIcon className="size-3" />
              {t("Appointment")}
            </span>
          )}
        </span>
        {parts.map((row, index) => (
          <span key={row.key} className={cn("dk-ex-part", index === 0 && "dk-lead")}>
            <ReadValue row={row} {...pick} />
          </span>
        ))}
        {extras.length > 0 && (
          <span className="dk-ex-extras">
            {extras.map((row) => (
              <span key={row.key}>
                <em>{row.label}</em>
                <ReadValue row={row} {...pick} />
              </span>
            ))}
          </span>
        )}
      </div>
    </li>
  );
}

/** A short column of label and value. */
function FactPanel({
  title,
  rows,
  ...pick
}: ValuePick & { title: string; rows: readonly FieldRow[] }) {
  if (rows.length === 0) return null;
  return (
    <div className="dk-ex-panel">
      <h4>{title}</h4>
      <dl>
        {rows.map((row) => {
          const money = row.kind === "read" && MONEY_KEYS.has(row.field.key);
          return (
            <div key={row.key} className={cn(money && "dk-money")}>
              <dt>{row.label}</dt>
              <dd>
                <ReadValue row={row} {...pick}>
                  {money && row.kind === "read" ? readMoney(row.field.value) : undefined}
                </ReadValue>
              </dd>
            </div>
          );
        })}
      </dl>
    </div>
  );
}

/** A value as read, which finds itself on the page; a value not read is plain text. */
function ReadValue({
  row,
  selected,
  hot,
  multiPage,
  onSelect,
  onPoint,
  children,
}: ValuePick & { row: FieldRow; children?: ReactNode }) {
  const t = useT();
  if (row.kind === "missing") return <span className="dk-ex-blank">{t("Not read")}</span>;
  const { field } = row;
  return (
    <Button
      variant="bare"
      size="bare"
      className={cn(
        "dk-ex-v ui-focus-ring",
        `dk-${row.state}`,
        selected === row.key && "dk-on",
        hot === row.key && "dk-hot",
      )}
      aria-pressed={selected === row.key}
      aria-label={rowAnnouncement(row, multiPage, t)}
      title={
        field.page > 0
          ? t("{0}% sure · page {1}", Math.round(field.confidence * 100), field.page)
          : t("{0}% sure", Math.round(field.confidence * 100))
      }
      onClick={() => onSelect(row.key)}
      onMouseEnter={() => onPoint(row.key)}
      onMouseLeave={() => onPoint(null)}
    >
      {children ?? field.value}
    </Button>
  );
}
