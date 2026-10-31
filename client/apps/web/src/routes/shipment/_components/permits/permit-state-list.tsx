import type { PermitChecklistEntry } from "@/lib/permit-checklist";
import {
  CheckIcon,
  DotsHorizontalIcon,
  Edit02Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { describeExceedance, requirementStateCode } from "@trenova/shared/lib/permit";
import { cn } from "@trenova/shared/lib/utils";
import type { Permit, PermitRequirement, PermitStatus } from "@trenova/shared/types/permit";
import type { ReactNode } from "react";

/** The state's two letters in a tile, drawn in the tone of where its permit stands. */
function StateTile({ code, phase }: { code: string; phase: PermitChecklistEntry["phase"] }) {
  return (
    <span
      aria-hidden
      className={cn(
        "relative flex size-8 shrink-0 items-center justify-center rounded-md border text-2xs font-semibold",
        phase === "todo" &&
          "border-warning-border bg-warning-subtle text-warning-subtle-foreground",
        phase === "filed" &&
          "border-success-border bg-success-subtle text-success-subtle-foreground",
        phase === "waived" && "bg-muted text-foreground-muted",
      )}
    >
      {code || "—"}
      {phase === "filed" && (
        <span className="bg-success text-foreground-on-solid ring-card animate-confirm absolute -right-1 -bottom-1 grid size-3.5 place-items-center rounded-full ring-2">
          <CheckIcon className="size-2" strokeWidth={3} />
        </span>
      )}
    </span>
  );
}

/** The parts of a row's line beneath its name, set apart by quiet dots. */
function Subline({ parts }: { parts: { key: string; node: ReactNode }[] }) {
  return (
    <span className="text-foreground-muted flex min-w-0 flex-wrap items-center gap-x-1.5 text-2xs">
      {parts.map((part, index) => (
        <span key={part.key} className="flex items-center gap-x-1.5">
          {index > 0 && <span aria-hidden>·</span>}
          {part.node}
        </span>
      ))}
    </span>
  );
}

const PERMIT_STATUS_LABELS: Record<PermitStatus, string> = defineLabels({
  Active: "Active",
  Pending: "Pending",
  Expired: "Expired",
  Void: "Void",
});

const PERMIT_STATUS_TEXT_TONE: Partial<Record<PermitStatus, string>> = {
  Pending: "text-warning-foreground",
  Expired: "text-danger",
  Void: "text-foreground-subtle",
};

function stateName(requirement: PermitRequirement | null, permit: Permit | null): string {
  return (
    requirement?.provenance?.stateName ||
    permit?.state?.name ||
    (requirement ? requirementStateCode(requirement) : permit?.state?.abbreviation) ||
    ""
  );
}

function StateRow({
  entry,
  onRecord,
  onWaive,
  onEdit,
}: {
  entry: PermitChecklistEntry;
  onRecord: (requirement: PermitRequirement) => void;
  onWaive: (requirement: PermitRequirement) => void;
  onEdit: (permit: Permit) => void;
}) {
  const t = useT();
  const { requirement, permit, phase } = entry;
  const code = requirement
    ? requirementStateCode(requirement)
    : (permit?.state?.abbreviation ?? "");
  const name = stateName(requirement, permit) || t("Permit");
  const exceedances = requirement?.exceedances ?? [];
  // A requirement derived since the last save has no row to record against yet.
  const actionable = phase === "todo" && !!requirement?.id;

  const parts: { key: string; node: ReactNode }[] = [];
  if (phase === "todo" && requirement) {
    parts.push({ key: "needs", node: t("Needs a permit") });
    parts.push({
      key: "lead",
      node: t("{0, plural, one {# day lead} other {# days lead}}", requirement.leadTimeDays),
    });
    if (requirement.validityDays > 0) {
      parts.push({
        key: "valid",
        node: t("{0, plural, one {valid # day} other {valid # days}}", requirement.validityDays),
      });
    }
    if (!requirement.id) {
      parts.push({ key: "save", node: t("Save the shipment to record it") });
    }
  } else if (phase === "waived" && requirement) {
    parts.push({
      key: "waived",
      node: requirement.waiverReason ? t("Waived: {0}", requirement.waiverReason) : t("Waived"),
    });
  } else if (permit) {
    parts.push({
      key: "number",
      node: <span className="text-foreground font-mono">{permit.permitNumber}</span>,
    });
    parts.push({
      key: "expires",
      node: permit.expiresAt
        ? t("Expires {0}", formatUnixDate(permit.expiresAt))
        : t("No expiry recorded"),
    });
    if (permit.status !== "Active") {
      parts.push({
        key: "status",
        node: (
          <span className={PERMIT_STATUS_TEXT_TONE[permit.status]}>
            {PERMIT_STATUS_LABELS[permit.status]}
          </span>
        ),
      });
    }
  } else {
    parts.push({ key: "filed", node: t("Permit recorded") });
  }

  const menu = actionable
    ? [
        <DropdownMenuItem
          key="waive"
          title={t("Waive")}
          description={t("Mark this state as not needing a permit, with a reason")}
          startContent={<XCloseIcon className="size-3.5" />}
          onClick={() => requirement && onWaive(requirement)}
        />,
      ]
    : permit
      ? [
          <DropdownMenuItem
            key="edit"
            title={t("Edit permit")}
            description={t("Correct its number, dates or status")}
            startContent={<Edit02Icon className="size-3.5" />}
            onClick={() => onEdit(permit)}
          />,
        ]
      : [];

  return (
    <li
      className="animate-rise flex items-center gap-3 py-2.5"
      data-permit-phase={phase}
      data-state={code}
    >
      <StateTile code={code} phase={phase} />
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
          <span
            className={cn(
              "truncate text-sm font-medium",
              phase === "waived" && "text-foreground-muted",
            )}
          >
            {name}
          </span>
          {requirement?.isSuperload && (
            <span className="bg-warning-subtle text-warning-subtle-foreground rounded-sm px-1.5 text-2xs font-medium">
              {t("Superload")}
            </span>
          )}
          {phase === "todo" &&
            exceedances.map((exceedance) => (
              <span
                key={exceedance.trigger}
                className="bg-muted text-foreground-muted rounded-sm px-1.5 text-2xs tabular-nums"
              >
                {describeExceedance(exceedance)}
              </span>
            ))}
        </span>
        <Subline parts={parts} />
      </div>
      {actionable && requirement && (
        <Button type="button" variant="outline" size="xxs" onClick={() => onRecord(requirement)}>
          {t("Record permit")}
        </Button>
      )}
      {menu.length > 0 && (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                type="button"
                variant="quiet"
                size="icon-xs"
                aria-label={t("{0} permit actions", name)}
              >
                <DotsHorizontalIcon className="size-3.5" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" sideOffset={4} className="w-72">
            {menu}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </li>
  );
}

/**
 * The checklist itself, a line per state: what is still to file, with its trigger, lead time and
 * the button to record the permit (waiving it in the menu), then the permits filed, with their
 * numbers and expiry, and the states waived.
 */
export function PermitStateList({
  entries,
  onRecord,
  onWaive,
  onEdit,
}: {
  entries: PermitChecklistEntry[];
  onRecord: (requirement: PermitRequirement) => void;
  onWaive: (requirement: PermitRequirement) => void;
  onEdit: (permit: Permit) => void;
}) {
  return (
    <ul className="divide-y">
      {entries.map((entry) => (
        <StateRow
          key={entry.key}
          entry={entry}
          onRecord={onRecord}
          onWaive={onWaive}
          onEdit={onEdit}
        />
      ))}
    </ul>
  );
}
