import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { PreloadIntent } from "@/components/work-pane";
import type { EmploymentVerificationRow } from "@/lib/graphql/worker-dqf";
import {
  Edit03Icon,
  RefreshCw01Icon,
  Send01Icon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRange, formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  verificationMethodLabel,
  verificationNextStep,
  verificationSettled,
  verificationStatusLabel,
} from "@trenova/shared/lib/dqf";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import { ATTENTION_CLASS, type FlashPhase } from "../../use-row-flash";
import {
  employerStops,
  employerTone,
  type EmployerStop,
  type EmployerStopKey,
} from "./dq-file-model";

export type EmployerPermissions = { canUpdate: boolean; canDelete: boolean };

/** The dates employed there: date-only fields, read in the reader's zone (Patterns 14). */
/** A past job's dates are read in months, as a reference check asks for them. */
const MONTH = { unit: "month", year: "always" } as const;

function employedRange(t: TranslateFn, verification: EmploymentVerificationRow): string | null {
  const { employedFrom, employedTo } = verification;
  if (employedFrom && employedTo) {
    return formatRange(employedFrom, employedTo, { ...MONTH, join: "words" });
  }
  if (employedFrom) return t("Since {0}", formatRange(employedFrom, employedFrom, MONTH));
  if (employedTo) return t("Until {0}", formatRange(employedTo, employedTo, MONTH));
  return null;
}

function stopName(t: TranslateFn, key: EmployerStopKey): string {
  switch (key) {
    case "recorded":
      return t("Not requested");
    case "requested":
      return t("Requested");
    case "chased":
      return t("Chased");
    case "answered":
      return t("Answered");
  }
}

/** How the request went, in a sentence under the line: who it went to and what came back. */
function reportLine(
  t: TranslateFn,
  verification: EmploymentVerificationRow,
  now: number,
): string | null {
  const next = verificationNextStep(verification, now);
  switch (verification.status) {
    case "Pending":
      return verification.contactEmail
        ? t("Not sent yet. It goes to {0}.", verification.contactEmail)
        : t("Not sent yet.");
    case "Requested": {
      const sent = verification.contactEmail
        ? t(
            "Requested by {0} to {1}",
            toSentenceFragment(verificationMethodLabel(verification.method)),
            verification.contactEmail,
          )
        : t("Requested by {0}", toSentenceFragment(verificationMethodLabel(verification.method)));
      if (next.action === "wait" && next.dueAt) {
        return t("{0}. Chase again from {1}.", sent, formatUnixDateMedium(next.dueAt));
      }
      if (next.action === "close") {
        return t(
          "{0}. No answer after {1, plural, one {# chase} other {# chases}}.",
          sent,
          verification.followUpCount,
        );
      }
      return t("{0}.", sent);
    }
    case "Received": {
      const accidents = verification.hadAccidents ? verification.accidentCount : 0;
      const violations = verification.wasDotRegulated && verification.hadDrugAlcoholViolations;
      const findings = !verification.wasDotRegulated
        ? t(
            "{0, plural, =0 {No accidents reported.} one {# accident reported.} other {# accidents reported.}}",
            accidents,
          )
        : violations
          ? t(
              "{0, plural, =0 {No accidents; drug or alcohol violations reported.} one {# accident and drug or alcohol violations reported.} other {# accidents and drug or alcohol violations reported.}}",
              accidents,
            )
          : t(
              "{0, plural, =0 {No accidents, no violations.} one {# accident, no violations.} other {# accidents, no violations.}}",
              accidents,
            );
      const history = verification.wasDotRegulated
        ? verification.drugAlcoholResponseReceivedAt
          ? t(
              "Drug and alcohol history received {0}.",
              formatUnixDateMedium(verification.drugAlcoholResponseReceivedAt),
            )
          : t("Drug and alcohol history not received yet (49 CFR 382.413).")
        : null;
      return history ? t("{0} {1}", findings, history) : findings;
    }
    default:
      return null;
  }
}

/**
 * One previous employer: the request drawn as four stops with the dates that prove each, how it
 * went in a sentence, and a menu to send, chase, edit or remove it.
 */
export function EmployerRow({
  verification,
  permissions,
  deadline,
  busy,
  flash,
  intent,
  onRequest,
  onFollowUp,
  onEdit,
  onRemove,
}: {
  verification: EmploymentVerificationRow;
  permissions: EmployerPermissions;
  deadline: { dueAt: number; late: boolean };
  busy: boolean;
  flash: FlashPhase | undefined;
  /** Reads the edit form's code as the pointer or focus reaches the row's menu. */
  intent?: PreloadIntent;
  onRequest: (verification: EmploymentVerificationRow) => void;
  onFollowUp: (verification: EmploymentVerificationRow) => void;
  onEdit: (verification: EmploymentVerificationRow) => void;
  onRemove: (verification: EmploymentVerificationRow) => void;
}) {
  const t = useT();
  const now = useNow("minute");
  const settled = verificationSettled(verification.status);
  const late = deadline.late && !settled;
  const tone = employerTone(verification, deadline.late);
  const stops = employerStops(t, verification, deadline, now);
  const range = employedRange(t, verification);
  const report = reportLine(t, verification, now);

  const actions: RowAction[] = [];
  if (permissions.canUpdate && verification.status === "Pending") {
    actions.push({
      id: "request",
      label: t("Send the request"),
      icon: Send01Icon,
      disabled: busy,
      onSelect: () => onRequest(verification),
    });
  }
  if (permissions.canUpdate && verification.status === "Requested") {
    actions.push({
      id: "chase",
      label: t("Chase again"),
      icon: RefreshCw01Icon,
      disabled: busy,
      onSelect: () => onFollowUp(verification),
    });
  }
  if (permissions.canUpdate) {
    actions.push({
      id: "edit",
      label: t("Edit"),
      icon: Edit03Icon,
      preload: intent?.onFocus,
      onSelect: () => onEdit(verification),
    });
  }
  if (permissions.canDelete && !settled) {
    actions.push({
      id: "remove",
      label: t("Remove"),
      icon: Trash01Icon,
      destructive: true,
      disabled: busy,
      onSelect: () => onRemove(verification),
    });
  }

  return (
    <div
      data-glide-target
      data-testid={`dqf-employer-${verification.id}`}
      data-state={tone}
      data-changed={flash}
      className={cn("relative rounded-md px-2 pt-2.5 pb-3", ATTENTION_CLASS)}
    >
      <div className="flex min-h-5.5 items-center gap-2">
        <span className="text-sm font-medium">{verification.employerName}</span>
        {late ? (
          <Tooltip>
            <TooltipTrigger
              render={
                <span tabIndex={0} className="rounded-badge ui-focus-ring">
                  <Badge variant="danger" dot>
                    {t("Late")}
                  </Badge>
                </span>
              }
            />
            <TooltipContent>
              {t("Was due {0}", formatUnixDateMedium(deadline.dueAt))}
            </TooltipContent>
          </Tooltip>
        ) : null}
        {verification.wasDotRegulated ? null : <Badge variant="neutral">{t("Non-DOT")}</Badge>}
        {range ? <span className="text-xs text-muted-foreground tabular-nums">{range}</span> : null}
        <RowActionsMenu
          className="ml-auto"
          label={t("Actions for {0}", verification.employerName)}
          actions={actions}
        />
      </div>
      <ProgressStops stops={stops} />
      <p className="mt-2 text-xs">
        <span
          className={cn(
            "font-medium",
            tone === "success" && "text-success-foreground",
            tone === "danger" && "text-danger-foreground",
            tone === "warning" && "text-warning-foreground",
          )}
        >
          {t("{0}.", verificationStatusLabel(verification.status))}
        </span>
        {report ? <span className="text-muted-foreground"> {report}</span> : null}
      </p>
      {verification.findings ? (
        <p className="mt-1 text-xs text-muted-foreground">{verification.findings}</p>
      ) : null}
    </div>
  );
}

function ProgressStops({ stops }: { stops: readonly EmployerStop[] }) {
  const t = useT();
  return (
    <ol
      aria-label={t("Safety history request progress")}
      className="mt-2 grid max-w-160 auto-cols-fr grid-flow-col overflow-x-auto"
    >
      {stops.map((stop, index) => {
        const last = index === stops.length - 1;
        return (
          <li
            key={stop.key}
            aria-current={stop.state === "current" ? "step" : undefined}
            data-state={stop.state}
            className="flex min-w-24 flex-col pr-2"
          >
            <span aria-hidden className="relative block h-3.5">
              <span
                className={cn(
                  "absolute top-1/2 left-0 -translate-y-1/2 rounded-full",
                  stop.state === "done" && "size-1.75 bg-foreground",
                  stop.state === "ahead" && "size-1.75 border-[1.5px] border-border-strong bg-card",
                  stop.state === "current" &&
                    "grid size-2.25 place-items-center border-[1.5px] bg-card ring-2 ring-border ring-offset-1 ring-offset-card",
                  stop.state === "current" &&
                    (stop.danger
                      ? "border-danger-foreground ring-danger-border"
                      : "border-foreground"),
                )}
              >
                {stop.state === "current" ? (
                  <span
                    className={cn(
                      "size-1 rounded-full",
                      stop.danger ? "bg-danger-foreground" : "bg-foreground",
                    )}
                  />
                ) : null}
              </span>
              {last ? null : (
                <span
                  className={cn(
                    "absolute top-1/2 right-1 left-3 h-0 origin-left",
                    stop.state === "done"
                      ? "animate-fill border-t-[1.5px] border-foreground motion-reduce:animate-none"
                      : "border-t border-dashed border-border-strong",
                  )}
                />
              )}
            </span>
            <span
              className={cn(
                "mt-1 truncate text-xs",
                stop.state === "ahead" ? "text-muted-foreground" : "text-foreground",
                stop.state === "current" && "font-medium",
              )}
            >
              {stopName(t, stop.key)}
            </span>
            {stop.meta ? (
              <span
                className={cn(
                  "truncate text-xs tabular-nums",
                  stop.state === "current" && stop.danger
                    ? "text-danger-foreground"
                    : stop.state === "ahead"
                      ? "text-foreground-subtle"
                      : "text-muted-foreground",
                )}
              >
                {stop.meta}
              </span>
            ) : null}
          </li>
        );
      })}
    </ol>
  );
}
