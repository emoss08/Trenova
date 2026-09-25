import type {
  AgentPreviewOperation,
  PreviewFieldChange,
  PreviewRef,
} from "@/lib/graphql/agent-preview";
import { DescriptionEmpty } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon, LockIcon } from "lucide-react";
import { Link } from "react-router";
import { formatDisplayValue, isFigureType, type DisplayType } from "../readable-values";
import { displayTypeOf, previewRecordPath } from "./preview-format";

/** The words a person reads for data they may not see, with the lock that says why. */
export function HiddenValue({ className }: { className?: string }) {
  const t = useT();

  return (
    <span
      className={cn("text-foreground-subtle inline-flex items-center gap-1 text-xs", className)}
    >
      <LockIcon aria-hidden className="size-3 shrink-0" />
      {t("Hidden by your data access")}
    </span>
  );
}

/** Operations whose values have no "before": the record, the message or the run is new. */
const WITHOUT_BEFORE: ReadonlySet<AgentPreviewOperation> = new Set(["Create", "Send", "Run"]);

/** Operations whose values have no "after": the record is going. */
const WITHOUT_AFTER: ReadonlySet<AgentPreviewOperation> = new Set(["Delete"]);

function isAbsent(value: unknown): boolean {
  return value === null || value === undefined || value === "";
}

/**
 * One side of a change. A value that points at a record reads as that
 * record's name, linked where the record has a page; the id it holds is never
 * shown. A reference the reader may not follow is hidden with the rest.
 */
function Side({
  value,
  reference,
  type,
  struck,
}: {
  value: unknown;
  reference: PreviewRef | null | undefined;
  type: DisplayType;
  struck: boolean;
}) {
  const t = useT();
  const tone = struck ? "text-foreground-subtle line-through" : "text-foreground";

  if (reference) {
    if (reference.withheld) {
      return <HiddenValue />;
    }
    const label = reference.label ?? "";
    if (label === "") {
      return <span className={cn(tone, "italic")}>{t("A record that is no longer there")}</span>;
    }
    const path = previewRecordPath(reference.record);
    return path !== null && !struck ? (
      <Link to={path} className="text-brand break-words hover:underline">
        {label}
      </Link>
    ) : (
      <span className={cn(tone, "break-words")}>{label}</span>
    );
  }

  const text = isAbsent(value) ? "" : formatDisplayValue(type, value, t);
  if (text === "") {
    return <DescriptionEmpty />;
  }

  return (
    <span
      className={cn(
        tone,
        "break-words",
        (isFigureType(type) || type === "date" || type === "datetime") && "font-mono tabular-nums",
        type === "longText" && "whitespace-pre-wrap",
      )}
    >
      {text}
    </span>
  );
}

/**
 * A value a write would set: the old one struck through in a quieter ink, an
 * arrow, the new one. A value the reader may not see says so rather than
 * disappearing, a value that moved since the proposal says what it was then,
 * and a value that starts from an earlier plan step says which.
 */
export function ValueChange({
  change,
  operation,
}: {
  change: PreviewFieldChange;
  operation: AgentPreviewOperation;
}) {
  const t = useT();
  const type = displayTypeOf(change.valueType);

  if (change.withheld) {
    return <HiddenValue />;
  }

  const showBefore = !WITHOUT_BEFORE.has(operation);
  const showAfter = !WITHOUT_AFTER.has(operation);
  const proposedText =
    change.changedSinceProposed && !isAbsent(change.proposedBefore)
      ? formatDisplayValue(type, change.proposedBefore, t)
      : "";

  return (
    <span className="flex min-w-0 flex-col gap-0.5">
      <span className="flex min-w-0 flex-wrap items-baseline gap-x-1.5 gap-y-0.5">
        {showBefore && (
          <Side value={change.before} reference={change.beforeRef} type={type} struck />
        )}
        {showBefore && showAfter && (
          <>
            <ArrowRightIcon
              aria-hidden
              className="text-foreground-subtle size-3 shrink-0 self-center"
            />
            <span className="sr-only">{t("becomes")}</span>
          </>
        )}
        {showAfter && (
          <Side value={change.after} reference={change.afterRef} type={type} struck={false} />
        )}
        {change.truncated && (
          <span className="text-foreground-subtle text-xs">{t("(shortened)")}</span>
        )}
      </span>
      {change.changedSinceProposed && (
        <span className="text-warning text-xs">
          {proposedText !== ""
            ? t("Was {0} when proposed", proposedText)
            : t("Was empty when proposed")}
        </span>
      )}
      {change.projectedFromStep > 0 && (
        <span className="text-foreground-subtle text-xs">
          {t("As step {0} leaves it", change.projectedFromStep)}
        </span>
      )}
      {change.volatile && (
        <span className="text-foreground-subtle text-xs">{t("May move before it runs")}</span>
      )}
    </span>
  );
}
