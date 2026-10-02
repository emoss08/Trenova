import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import {
  runDiffFrom,
  type RunDiffChangeArtifact,
  type RunDiffMeasureArtifact,
} from "./artifact-payloads";
import { ArtifactScroll, ArtifactSection } from "./artifact-section";

const RUN_STAMP = { month: "short", day: "numeric", hour: "numeric", minute: "2-digit" } as const;

/**
 * What a change in a row is, as a tone. This is a severity ordering, not a set
 * of categories: a repeated key means the figures for that row cannot be
 * trusted at all, which is worse news than any movement in them.
 */
const CHANGE_TONE: Record<string, BadgeVariant> = {
  Duplicate: "danger",
  Removed: "warning",
  Added: "success",
  Changed: "neutral",
  Unchanged: "neutral",
};

function changeLabel(kind: string, t: (s: string) => string): string {
  switch (kind) {
    case "Duplicate":
      return t("Repeated key");
    case "Removed":
      return t("Gone");
    case "Added":
      return t("New");
    case "Changed":
      return t("Changed");
    default:
      return t("Unchanged");
  }
}

/** A signed figure reads as a direction; an unsigned one has to be read twice. */
function deltaClass(delta: string): string {
  if (delta.startsWith("-")) {
    return "text-danger";
  }
  if (delta === "" || delta === "0") {
    return "text-muted-foreground";
  }

  return "text-success";
}

function Movement({ measure }: { measure: RunDiffMeasureArtifact }) {
  return (
    <span className="tabular-nums">
      <span className="text-muted-foreground">{measure.before}</span>
      <span aria-hidden className="text-muted-foreground px-1">
        →
      </span>
      <span>{measure.after}</span>
      <span className={cn("pl-1.5", deltaClass(measure.delta))}>{measure.delta}</span>
    </span>
  );
}

function ChangeRow({ change, t }: { change: RunDiffChangeArtifact; t: (s: string) => string }) {
  const variant = CHANGE_TONE[change.kind] ?? "neutral";

  return (
    <li className="flex flex-col gap-1 px-3 py-2">
      <div className="flex items-start justify-between gap-2">
        <span className="text-xs">{change.keyValues.filter(Boolean).join(" · ")}</span>
        <Badge variant={variant}>{changeLabel(change.kind, t)}</Badge>
      </div>

      {/* A repeated key carries no measures on purpose: the run never had one
          row's numbers for it, so there is nothing honest to show. */}
      {change.measures.map((measure) => (
        <div key={measure.column} className="flex items-baseline justify-between gap-2 text-xs">
          <span className="text-muted-foreground">{measure.label}</span>
          {change.kind === "Changed" ? (
            <Movement measure={measure} />
          ) : (
            <span className="tabular-nums">{measure.after}</span>
          )}
        </div>
      ))}
    </li>
  );
}

/**
 * What moved between two runs of the same report.
 *
 * Reading the newest run alone cannot answer "what changed since last week" —
 * it has no record of what is no longer in it. Here the two runs are named at
 * the top so a percentage has two dates behind it, the rows that cannot be
 * trusted come first, then what disappeared, then what is new, then what moved
 * with the biggest movement leading, and the totals for both sides sit
 * underneath.
 */
export function RunDiffArtifact({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const diff = useMemo(() => runDiffFrom(artifact), [artifact]);

  const counts = [
    { label: t("Added"), value: diff.counts.added },
    { label: t("Gone"), value: diff.counts.removed },
    { label: t("Changed"), value: diff.counts.changed },
    { label: t("Unchanged"), value: diff.counts.unchanged },
    { label: t("Repeated key"), value: diff.counts.duplicate },
  ].filter((count) => count.value > 0);

  return (
    <ArtifactScroll>
      <div className="space-y-0.5">
        <p className="text-sm font-semibold">
          {diff.after.reportName !== "" ? diff.after.reportName : t("Report")}
        </p>
        <p className="text-muted-foreground text-xs">
          {t(
            "{0} compared with {1}",
            formatUnixInUserTimezone(diff.after.generatedAt, RUN_STAMP),
            formatUnixInUserTimezone(diff.before.generatedAt, RUN_STAMP),
          )}
        </p>
        <p className="text-muted-foreground text-xs">{t("Matched on {0}", diff.keys.join(", "))}</p>
      </div>

      {counts.length > 0 && (
        <ul className="grid grid-cols-3 gap-2">
          {counts.map((count) => (
            <li key={count.label} className="bg-sunken flex flex-col rounded-lg px-3 py-2">
              <span className="text-base font-semibold tabular-nums">{count.value}</span>
              <span className="text-muted-foreground text-xs">{count.label}</span>
            </li>
          ))}
        </ul>
      )}

      {/* A row cap on either side means a row reported gone may simply be past
          it, which changes what the answer means. */}
      {diff.truncated && (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t(
              "One of these runs hit its row cap, so a row shown as gone may only be past the cap.",
            )}
          </AlertDescription>
        </Alert>
      )}

      <ArtifactSection
        title={t("Changes")}
        hint={diff.changes.length > 0 ? diff.changes.length : undefined}
        inset={false}
      >
        {diff.changes.length === 0 ? (
          <p className="text-muted-foreground px-3 py-2.5 text-xs">
            {diff.note !== "" ? diff.note : t("Nothing moved between these two runs.")}
          </p>
        ) : (
          <ul className="divide-border-subtle flex flex-col divide-y">
            {diff.changes.map((change, index) => (
              <ChangeRow key={`${change.key}-${index}`} change={change} t={t} />
            ))}
          </ul>
        )}
      </ArtifactSection>

      {diff.totals.length > 0 && (
        <ArtifactSection title={t("Totals")} inset={false}>
          <ul className="divide-border-subtle flex flex-col divide-y">
            {diff.totals.map((total) => (
              <li
                key={total.column}
                className="flex items-baseline justify-between gap-2 px-3 py-1.5 text-xs"
              >
                <span className="text-muted-foreground">{total.label}</span>
                <Movement measure={total} />
              </li>
            ))}
          </ul>
        </ArtifactSection>
      )}
    </ArtifactScroll>
  );
}
