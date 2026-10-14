import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { EDIInspectionDiagnostic } from "@trenova/shared/types/edi";
import { CardIcon } from "./record-card";

/** How a finding reads: an error, a warning, or a note. */
export function findingSeverity(diagnostic: EDIInspectionDiagnostic): "bad" | "warn" | "info" {
  return diagnostic.severity === "Error"
    ? "bad"
    : diagnostic.severity === "Warning"
      ? "warn"
      : "info";
}

/** What the validator said of a segment, pinned under it, with the fix it suggests. */
export function EdiFinding({ diagnostic }: { diagnostic: EDIInspectionDiagnostic }) {
  const t = useT();
  const severity = findingSeverity(diagnostic);
  const where =
    diagnostic.elementPosition > 0
      ? t("{0}, element {1}", diagnostic.segmentId ?? "", formatNumber(diagnostic.elementPosition))
      : "";
  return (
    <div className={cn("dk-edi-mark", `dk-${severity}`)}>
      <span className="dk-edi-mark-i" aria-hidden>
        <CardIcon
          name={severity === "bad" ? "x" : severity === "warn" ? "warn" : "info"}
          size={10}
          stroke={2.8}
        />
      </span>
      <span className="dk-edi-mark-b">
        <b>{diagnostic.message}</b>
        {(where || diagnostic.code) && (
          <span className="dk-edi-mark-at">
            {where && <span className="dk-edi-mark-el">{where}</span>}
            {diagnostic.code && <code>{diagnostic.code}</code>}
          </span>
        )}
        {diagnostic.suggestedFix && (
          <span className="dk-edi-mark-fix">
            <CardIcon name="forward" size={11} stroke={2.4} />
            <span>{t("Fix: {0}", diagnostic.suggestedFix)}</span>
          </span>
        )}
      </span>
    </div>
  );
}
