import { PreviewRemedies } from "@/components/assistant/proposal-preview/preview-remedies";
import {
  previewWarningText,
  previewWarningTone,
  reasonSentence,
  visibleWarnings,
  wouldFailReasons,
  type WouldFailReason,
} from "@/components/assistant/proposal-preview/preview-warnings";
import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type Tone = "danger" | "warning" | "info";

/**
 * What to know before deciding a change, said once, above the change itself:
 * that its record moved, why it would be refused with the value to change
 * beside each reason, and every warning with its fix where there is one. The
 * block takes the tone of the worst thing in it, and is not drawn at all when
 * there is nothing to say.
 */
export function DecisionNotices({
  preview,
  canChange,
  onChange,
}: {
  preview: ProposalPreview;
  /** Whether the person may change the value a refusal names. */
  canChange?: (param: string) => boolean;
  /** Opens the editor on that value. */
  onChange?: (reason: WouldFailReason) => void;
}) {
  const t = useT();
  const warnings = visibleWarnings(preview.warnings, { inPlan: false });
  const notes: { key: string; tone: Tone; body: ReactNode; action?: ReactNode }[] = [];

  if (preview.stale) {
    notes.push({
      key: "stale",
      tone: "warning",
      body: (
        <>
          <b className="font-medium text-dsk-fg">{t("Changed since it was proposed")}</b>{" "}
          {preview.staleness?.missing === true
            ? t("The record it would change is gone. It can only be rejected now.")
            : t(
                "The record it would change has been edited since. It can only be rejected now; ask the agent again for a fresh proposal.",
              )}
        </>
      ),
    });
  }
  warnings.forEach((warning, index) => {
    const reasons = wouldFailReasons(warning);
    if (reasons.length === 0) {
      notes.push({
        key: `${warning.code}-${index}`,
        tone: previewWarningTone(warning),
        body: (
          <>
            {previewWarningText(warning, t)}
            <PreviewRemedies warning={warning} />
          </>
        ),
      });
      return;
    }
    reasons.forEach((reason, at) => {
      const changeable =
        reason.param !== "" && onChange !== undefined && (canChange?.(reason.param) ?? false);
      notes.push({
        key: `${warning.code}-${index}-${at}`,
        tone: "danger",
        body: (
          <>
            {reason.label !== "" && <b className="font-medium text-dsk-fg">{reason.label}: </b>}
            {reasonSentence(reason, t)}
          </>
        ),
        action: changeable && reason.label !== "" && (
          <Button size="sm" variant="secondary" onClick={() => onChange?.(reason)}>
            {t("Change {0}", reason.label)}
          </Button>
        ),
      });
    });
  });

  if (notes.length === 0) {
    return null;
  }
  const blocked = notes.some((note) => note.tone === "danger");

  return (
    <section
      aria-label={blocked ? t("This can't run as it stands") : t("Before you decide")}
      className={cn(
        "flex flex-col rounded-[10px] border",
        blocked
          ? "border-danger-border bg-danger-subtle"
          : "border-warning-border bg-warning-subtle",
      )}
    >
      {notes.map((note) => (
        <div
          key={note.key}
          role={note.tone === "danger" ? "alert" : "status"}
          className={cn(
            "flex items-start gap-2.5 border-t px-3 py-2.5 text-sm first:border-t-0",
            blocked ? "border-danger-border" : "border-warning-border",
          )}
        >
          <span
            aria-hidden
            className={cn(
              "mt-1.75 size-1.5 shrink-0 rounded-full",
              note.tone === "danger"
                ? "bg-danger"
                : note.tone === "warning"
                  ? "bg-warning"
                  : "bg-dsk-subtle",
            )}
          />
          <div className="min-w-0 flex-1 break-words text-dsk-fg2">{note.body}</div>
          {note.action}
        </div>
      ))}
    </section>
  );
}
