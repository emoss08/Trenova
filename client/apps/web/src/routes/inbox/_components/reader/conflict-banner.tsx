import type {
  InboundMessageDetail,
  TriageConflict,
  TriageConflictResolution,
} from "@/lib/graphql/inbox";
import { AlertTriangleIcon, CheckIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay } from "@trenova/shared/lib/date";
import { formatCurrencyWhole, toSentenceFragment } from "@trenova/shared/lib/utils";
import { attachmentKindLabel } from "../classification";
import { useTriageActions } from "../use-triage-actions";

type ConflictCopy = { title: string; detail: string };

function documentName(t: TranslateFn, message: InboundMessageDetail, conflict: TriageConflict) {
  const attachment = message.attachments.find((row) => row.id === conflict.attachmentId);
  if (attachment === undefined) {
    return toSentenceFragment(t("Document"));
  }
  const read = attachment.reading?.documentType.trim() ?? "";

  return toSentenceFragment(read !== "" ? read : attachmentKindLabel(t, attachment.kind));
}

function amountLine(t: TranslateFn, conflict: TriageConflict, filed: string): string {
  const before = conflict.existingAmount === null ? Number.NaN : Number(conflict.existingAmount);
  const after = conflict.newAmount === null ? Number.NaN : Number(conflict.newAmount);
  if (Number.isNaN(before) || Number.isNaN(after)) {
    return filed === "" ? t("One is already on file.") : t("Filed {0}.", filed);
  }
  const difference = formatCurrencyWhole(Math.abs(after - before));
  const where = filed === "" ? t("On file at {0}.", formatCurrencyWhole(before)) : t("Filed {0} at {1}.", filed, formatCurrencyWhole(before));
  if (after === before) {
    return t("{0} This one says the same.", where);
  }

  return after > before
    ? t("{0} This one says {1} — {2} more.", where, formatCurrencyWhole(after), difference)
    : t("{0} This one says {1} — {2} less.", where, formatCurrencyWhole(after), difference);
}

/** A conflict's title and the line under it, from what the server found. */
export function conflictCopy(
  t: TranslateFn,
  message: InboundMessageDetail,
  conflict: TriageConflict,
): ConflictCopy {
  const pro = conflict.existingShipment?.proNumber ?? "";
  switch (conflict.kind) {
    case "DuplicateDocument": {
      const name = documentName(t, message, conflict);
      const filed =
        conflict.existingFiledAt === null ? "" : formatUnixMonthDay(conflict.existingFiledAt);
      return {
        title:
          pro === ""
            ? t("A {0} is already on file", name)
            : t("A {0} for {1} is already on file", name, pro),
        detail: amountLine(t, conflict, filed),
      };
    }
    case "OverlappingLoad": {
      const shipment = conflict.existingShipment;
      const load = conflict.loadIndex ?? 1;
      return {
        title: t("Load {0} may already exist", load),
        detail:
          shipment === null
            ? t("A load already on the board has the same lane and day.")
            : t(
                "{0} is {1} → {2} on the same day.",
                shipment.proNumber,
                shipment.originCity,
                shipment.destinationCity,
              ),
      };
    }
  }
}

function optionLabel(
  t: TranslateFn,
  resolution: TriageConflictResolution,
  conflict: TriageConflict,
): string {
  switch (resolution) {
    case "ReplaceExisting":
      return t("Replace the old one");
    case "KeepBoth":
      return t("Keep both");
    case "SkipDuplicate":
      return t("Skip load {0}", conflict.loadIndex ?? 1);
    case "ConfirmNew":
      return t("It's a new load");
  }
}

function resolvedLine(t: TranslateFn, conflict: TriageConflict, documentLabel: string): string {
  const pro = conflict.existingShipment?.proNumber ?? "";
  switch (conflict.resolution) {
    case "ReplaceExisting":
      return pro === ""
        ? t("Replacing the {0} on file", documentLabel)
        : t("Replacing the {0} on {1}", documentLabel, pro);
    case "KeepBoth":
      return t("Keeping both");
    case "SkipDuplicate":
      return pro === ""
        ? t("Skipping load {0}", conflict.loadIndex ?? 1)
        : t("Skipping load {0} — it's {1}", conflict.loadIndex ?? 1, pro);
    case "ConfirmNew":
      return t("Load {0} is new", conflict.loadIndex ?? 1);
    case null:
      return "";
  }
}

/**
 * What clashes with what is on file, with the choices that settle it; once
 * settled, one line saying how, and Undo.
 */
export function ConflictBanners({ message }: { message: InboundMessageDetail }) {
  const t = useT();
  const actions = useTriageActions();
  if (!message.needsReview || message.conflicts.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2">
      {message.conflicts.map((conflict) => {
        if (conflict.resolution !== null) {
          return (
            <div
              key={conflict.id}
              className="text-success animate-rise flex items-center gap-2 text-sm"
            >
              <CheckIcon aria-hidden className="size-3 shrink-0" strokeWidth={2.4} />
              <span className="min-w-0 flex-1">
                {resolvedLine(t, conflict, documentName(t, message, conflict))}
              </span>
              <button
                type="button"
                className="text-foreground-subtle hover:text-foreground ui-focus-ring rounded-sm text-xs"
                onClick={() => actions.reopenConflict(message.id, conflict.id)}
              >
                {t("Undo")}
              </button>
            </div>
          );
        }
        const copy = conflictCopy(t, message, conflict);
        return (
          <div
            key={conflict.id}
            role="alert"
            className="bg-warning-subtle border-warning-border animate-rise flex flex-col gap-1.5 rounded-lg border px-3 py-2.75"
          >
            <div className="text-warning-subtle-foreground flex items-center gap-1.75 text-base">
              <AlertTriangleIcon aria-hidden className="size-3.25 shrink-0" />
              <span className="font-semibold">{copy.title}</span>
            </div>
            <p className="text-foreground-muted text-sm leading-normal">{copy.detail}</p>
            <div className="mt-1 flex flex-wrap gap-1">
              {conflict.options.map((resolution, index) => (
                <Button
                  key={resolution}
                  size="sm"
                  variant={index === 0 ? "outline" : "ghost"}
                  onClick={() => actions.resolveConflict(message.id, conflict.id, resolution)}
                >
                  {optionLabel(t, resolution, conflict)}
                </Button>
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}
