import { useT } from "@trenova/shared/i18n/use-t";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { EmploymentVerificationRow } from "@/lib/graphql/worker-dqf";
import { Badge } from "@trenova/shared/components/ui/badge";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  verificationMethodLabel,
  verificationNextStep,
  verificationSettled,
  verificationStatusLabel,
  verificationTone,
} from "@trenova/shared/lib/dqf";
import { cn } from "@trenova/shared/lib/utils";
import { PencilIcon, RepeatIcon, SendIcon, Trash2Icon } from "lucide-react";

export type EmployerPermissions = { canUpdate: boolean; canDelete: boolean };

type EmployerRowProps = {
  verification: EmploymentVerificationRow;
  permissions: EmployerPermissions;
  now: number;
  busy: boolean;
  onRequest: (verification: EmploymentVerificationRow) => void;
  onFollowUp: (verification: EmploymentVerificationRow) => void;
  onEdit: (verification: EmploymentVerificationRow) => void;
  onDelete: (verification: EmploymentVerificationRow) => void;
};

const STAGES = ["Recorded", "Requested", "Answered"] as const;

function stageIndex(status: string): number {
  switch (status) {
    case "Pending":
      return 0;
    case "Requested":
      return 1;
    default:
      return 2;
  }
}

/**
 * One previous employer as a three-stage line — recorded, requested,
 * answered — with the dates that prove each stage and the next thing to do.
 * A silent employer reaches the end too: the chases on record are the answer.
 */
export function EmployerRow({
  verification,
  permissions,
  now,
  busy,
  onRequest,
  onFollowUp,
  onEdit,
  onDelete,
}: EmployerRowProps) {
  const t = useT();

  const next = verificationNextStep(verification, now);
  const stage = stageIndex(verification.status);
  const settled = verificationSettled(verification.status);

  const actions: RowAction[] = [];
  if (permissions.canUpdate && next.action === "request") {
    actions.push({
      id: "request",
      label: "Send the request",
      icon: SendIcon,
      disabled: busy,
      onSelect: () => onRequest(verification),
    });
  }
  if (permissions.canUpdate && verification.status === "Requested") {
    actions.push({
      id: "chase",
      label: "Chase again",
      icon: RepeatIcon,
      disabled: busy,
      onSelect: () => onFollowUp(verification),
    });
  }
  if (permissions.canUpdate) {
    actions.push({
      id: "edit",
      label:
        next.action === "close"
          ? "Close as no response"
          : next.action === "drugAlcohol"
            ? "Record the drug and alcohol history"
            : `Edit ${verification.employerName}`,
      icon: PencilIcon,
      onSelect: () => onEdit(verification),
    });
  }
  if (permissions.canDelete && !settled) {
    actions.push({
      id: "delete",
      label: `Remove ${verification.employerName}`,
      icon: Trash2Icon,
      destructive: true,
      disabled: busy,
      onSelect: () => onDelete(verification),
    });
  }

  const period =
    verification.employedFrom || verification.employedTo
      ? `${verification.employedFrom ? formatUnixDate(verification.employedFrom) : "?"} – ${
          verification.employedTo ? formatUnixDate(verification.employedTo) : "present"
        }`
      : null;
  const meta = [
    period,
    `by ${verificationMethodLabel(verification.method).toLowerCase()}`,
    verification.requestedAt ? `requested ${formatUnixDate(verification.requestedAt)}` : null,
    verification.followUpCount > 0
      ? `${verification.followUpCount} chase${verification.followUpCount === 1 ? "" : "s"}`
      : null,
    verification.responseReceivedAt
      ? `answered ${formatUnixDate(verification.responseReceivedAt)}`
      : null,
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <li
      data-testid={`dqf-employer-${verification.id}`}
      className="flex items-start gap-3 px-3 py-3"
    >
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium">{verification.employerName}</span>
          <Badge variant={verificationTone(verification.status)}>
            {verificationStatusLabel(verification.status)}
          </Badge>
          {verification.wasDotRegulated ? null : <Badge variant="secondary">{t("Non-DOT")}</Badge>}
        </div>

        <ol className="flex items-center gap-2" aria-label={t("Investigation progress")}>
          {STAGES.map((label, index) => {
            const done = index < stage || (index === stage && settled);
            const current = index === stage && !settled;
            return (
              <li key={label} className="flex flex-1 items-center gap-1.5 last:flex-none">
                <span
                  className={cn(
                    "size-2 shrink-0 rounded-full border",
                    done && "border-primary bg-primary",
                    current && "border-primary",
                    !done && !current && "border-border",
                  )}
                  aria-hidden
                />
                <span
                  className={cn(
                    "text-2xs uppercase",
                    done || current ? "text-foreground" : "text-muted-foreground",
                  )}
                >
                  {label}
                </span>
                {index < STAGES.length - 1 ? (
                  <span
                    className={cn("h-px flex-1", index < stage ? "bg-primary/40" : "bg-border")}
                    aria-hidden
                  />
                ) : null}
              </li>
            );
          })}
        </ol>

        <p className="text-muted-foreground text-xs">{meta}</p>

        {next.action === "wait" && next.dueAt ? (
          <p className="text-muted-foreground text-xs">
            {t("Waiting on the employer · chase again from {0}", formatUnixDate(next.dueAt))}
          </p>
        ) : next.action === "close" ? (
          <p className="text-xs">
            {t("No answer after {0} chases. The good-faith effort is on record; close it as no response.", verification.followUpCount)}
          </p>
        ) : next.action === "drugAlcohol" ? (
          <p className="text-xs">{t("Answered without the drug and alcohol history (49 CFR 382.413).")}</p>
        ) : null}

        {verification.hadAccidents || verification.hadDrugAlcoholViolations ? (
          <p className="text-xs">
            {verification.hadAccidents
              ? `${verification.accidentCount} accident${verification.accidentCount === 1 ? "" : "s"} reported. `
              : ""}
            {verification.hadDrugAlcoholViolations ? "Drug or alcohol violations reported." : ""}
          </p>
        ) : null}
        {verification.findings ? (
          <p className="text-muted-foreground text-xs">{verification.findings}</p>
        ) : null}
      </div>
      <RowActionsMenu label={`Actions for ${verification.employerName}`} actions={actions} />
    </li>
  );
}
