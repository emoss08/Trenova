import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionEmpty } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import {
  formatDisplayValue,
  formatMetric,
  humanizeCode,
  statusTone,
  type DisplayLink,
  type DisplayMetric,
  type DisplayType,
} from "./readable-values";

const WRITTEN_DATE = /^\d{4}-\d{2}-\d{2}/;

/**
 * One projected value, drawn the way its type reads: a status as a badge whose
 * tone follows its phase, a date in the reader's own timezone, money and
 * figures in tabular numerals, measurements as label and value, links as
 * links. An absent value is an em dash, never a blank a reader has to wonder
 * about.
 *
 * `inline` keeps a list of measurements or links on one line for a table cell;
 * elsewhere they wrap.
 */
export function DisplayValue({
  type,
  value,
  label,
  inline = false,
}: {
  type: DisplayType;
  value: unknown;
  /** The column's label; a flag that holds is said in its own words. */
  label: string;
  inline?: boolean;
}) {
  const t = useT();

  if (value === undefined || value === null || value === "") {
    return <DescriptionEmpty />;
  }

  switch (type) {
    case "status":
      return typeof value === "string" ? (
        <Badge variant={statusTone(value)}>{humanizeCode(value)}</Badge>
      ) : (
        <DescriptionEmpty />
      );
    case "flag":
      return <Badge variant="warning">{label}</Badge>;
    case "enum":
      return <span>{typeof value === "string" ? humanizeCode(value) : ""}</span>;
    case "boolean":
      return (
        <span className={cn(value === false && "text-foreground-muted")}>
          {formatDisplayValue(type, value, t)}
        </span>
      );
    case "date":
    case "datetime":
      if (typeof value === "number") {
        return (
          <time
            dateTime={new Date(value * 1000).toISOString()}
            title={formatUnixDateTimeMedium(value)}
            className="tabular-nums"
          >
            {formatDisplayValue(type, value, t)}
          </time>
        );
      }
      // A phrase in place of a date — "none on file" — is the absence said in
      // words, so it reads quieter than a date.
      return (
        <span
          className={cn(
            "tabular-nums",
            !WRITTEN_DATE.test(String(value)) && "text-foreground-muted",
          )}
        >
          {String(value)}
        </span>
      );
    case "money":
    case "number":
    case "percent":
      return <span className="tabular-nums">{formatDisplayValue(type, value, t)}</span>;
    case "metrics":
      return <Metrics metrics={value as DisplayMetric[]} inline={inline} />;
    case "links":
      return <Links links={value as DisplayLink[]} inline={inline} />;
    case "longText":
      return (
        <span className={cn(inline ? "truncate" : "leading-relaxed whitespace-pre-wrap")}>
          {String(value)}
        </span>
      );
    case "text":
      return <span>{formatDisplayValue(type, value, t)}</span>;
  }
}

function Metrics({ metrics, inline }: { metrics: DisplayMetric[]; inline: boolean }) {
  return (
    <span className={cn("flex min-w-0 gap-x-3 gap-y-0.5", inline ? "items-baseline" : "flex-wrap")}>
      {metrics.map((metric, index) => (
        <span
          key={`${metric.label}-${index}`}
          className="inline-flex shrink-0 items-baseline gap-1 whitespace-nowrap"
        >
          <span className="text-foreground-subtle">{metric.label}</span>
          <span className="tabular-nums">{formatMetric(metric)}</span>
        </span>
      ))}
    </span>
  );
}

function Links({ links, inline }: { links: DisplayLink[]; inline: boolean }) {
  return (
    <span className={cn("flex min-w-0 gap-x-3 gap-y-0.5", !inline && "flex-wrap")}>
      {links.map((link) => (
        <Link
          key={`${link.path}-${link.label}`}
          to={link.path}
          className="ui-focus-ring text-brand inline-flex shrink-0 items-baseline gap-1 rounded-control whitespace-nowrap underline-offset-2 hover:underline"
        >
          {link.label}
          {link.count > 0 && (
            <span className="text-foreground-subtle tabular-nums">{link.count}</span>
          )}
        </Link>
      ))}
    </span>
  );
}
