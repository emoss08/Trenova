import type { AssistantArtifact } from "@/types/assistant";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { runDiffFrom } from "../../artifact-payloads";

/** What moved between two runs of a report: the counts, the totals and each change. */
export function DeskDiffBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const diff = useMemo(() => runDiffFrom(artifact), [artifact]);
  const sign: Record<string, string> = { added: "+", removed: "−", changed: "~" };
  const when = (at: number) =>
    at > 0 ? new Date(at * 1000).toLocaleDateString([], { month: "short", day: "numeric" }) : "";

  return (
    <div className="dk-ax-pad dk-ax-diff">
      <div className="dk-ax-dh">
        <span>
          {diff.before.reportName || t("Before")} {when(diff.before.generatedAt)}
        </span>
        <i>→</i>
        <span>
          {diff.after.reportName || t("After")} {when(diff.after.generatedAt)}
        </span>
      </div>
      <div className="dk-ax-dc">
        <span className="dk-add">{t("+{0} added", diff.counts.added)}</span>
        <span className="dk-rem">{t("−{0} removed", diff.counts.removed)}</span>
        <span className="dk-chg">{t("~{0} changed", diff.counts.changed)}</span>
        <span>{t("{0} unchanged", diff.counts.unchanged)}</span>
      </div>
      {diff.totals.length > 0 && (
        <div className="dk-ax-dt">
          {diff.totals.map((measure) => (
            <div key={measure.column}>
              <span>{measure.label}</span>
              <b className="dk-ax-num">{measure.after}</b>
              <em
                className={cn(
                  measure.delta.startsWith("-") || measure.delta.startsWith("−")
                    ? "dk-dn"
                    : measure.delta !== "" && measure.delta !== "0"
                      ? "dk-up"
                      : "",
                )}
              >
                {measure.delta === "" || measure.delta === "0" ? t("no change") : measure.delta}
              </em>
            </div>
          ))}
        </div>
      )}
      <div className="dk-ax-dl">
        {diff.changes.map((change, index) => {
          const measure = change.measures[0];
          return (
            <div
              key={change.key + index}
              className={cn("dk-ax-dr", `dk-${change.kind}`)}
              style={{ animationDelay: `${index * 50}ms` }}
            >
              <span className="dk-ax-dk">{sign[change.kind] ?? "·"}</span>
              <span className="dk-ax-dn">
                <b>{change.keyValues[0] ?? change.key}</b>
                <span>{change.keyValues.slice(1).join(" · ")}</span>
              </span>
              <span className="dk-ax-num dk-mut">{measure?.before || "—"}</span>
              <i>→</i>
              <span className="dk-ax-num">{measure?.after || "—"}</span>
            </div>
          );
        })}
      </div>
      {diff.note !== "" && <p className="dk-ax-note">{diff.note}</p>}
    </div>
  );
}
