import { formatDisplayValue } from "@/components/assistant/readable-values";
import type { AssistantArtifact } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  analysisFrom,
  portalTaskFrom,
  boardFrom,
  emailDraftFrom,
  entityCardFrom,
  planFrom,
  rateExplanationFrom,
  runDiffFrom,
} from "./artifact-payloads";
import { connectionDraftFrom, draftOutline } from "./kinds/edi-connection-draft/connection-draft";
import { gridOf } from "./kinds/table/desk-table-body";

function money(amount: string, currency: string): string {
  const figure = Number(amount);
  if (amount === "" || !Number.isFinite(figure)) return "";
  return new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency || "USD",
    minimumFractionDigits: 2,
  }).format(figure);
}

/**
 * The line under an artifact's title in the list of everything a
 * conversation made: what is in it at a glance, the way the design writes
 * it. A table says how many rows, a record its status and who it is for, a
 * rate its total and the agreement that won, an email who it goes to. Empty
 * when there is nothing short worth saying.
 */
export function artifactPreview(artifact: AssistantArtifact, t: TranslateFn): string {
  switch (artifact.kind) {
    case "table_view":
      if ("path" in artifact.payload) return "";
      return t("{0, plural, one {# row} other {# rows}}", gridOf(artifact).rowCount);
    case "report_preview":
      return t("{0, plural, one {# row} other {# rows}}", gridOf(artifact).rowCount);
    case "entity_card": {
      const view = entityCardFrom(artifact).view;
      if (!view) return "";
      const status = view.status ? formatDisplayValue("status", view.status, t) : "";
      return [status, view.subtitle].filter((part) => part !== "").join(" · ");
    }
    case "rate_explanation": {
      const rate = rateExplanationFrom(artifact);
      return [
        money(rate.totals.total, rate.currency),
        rate.winner?.agreementName ?? rate.pricedBy?.method ?? "",
      ]
        .filter((part) => part !== "")
        .join(" · ");
    }
    case "email_draft": {
      const count = emailDraftFrom(artifact).to.length;
      return count > 0 ? t("{0, plural, one {# recipient} other {# recipients}}", count) : "";
    }
    case "plan": {
      const count = planFrom(artifact).stepCount;
      return count > 0 ? t("{0, plural, one {# step} other {# steps}}", count) : "";
    }
    case "run_diff": {
      const { counts } = runDiffFrom(artifact);
      return [
        counts.changed > 0 ? t("{0} changed", counts.changed) : "",
        counts.added > 0 ? t("{0} added", counts.added) : "",
        counts.removed > 0 ? t("{0} removed", counts.removed) : "",
      ]
        .filter((part) => part !== "")
        .join(" · ");
    }
    case "edi_connection_draft": {
      const draft = connectionDraftFrom(artifact);
      const outline = draftOutline(draft);
      return [
        draft.ready
          ? t("Ready to approve")
          : t(
              "{0, plural, one {# thing to settle} other {# things to settle}}",
              Math.max(draft.blockers.length, outline.decisions),
            ),
        draft.changes > 0 ? t("{0, plural, one {# change} other {# changes}}", draft.changes) : "",
      ]
        .filter((part) => part !== "")
        .join(" · ");
    }
    case "analysis": {
      const analysis = analysisFrom(artifact);
      if (analysis.error) return analysis.error.name;
      return analysis.charts.length > 0
        ? t("{0, plural, one {# chart} other {# charts}}", analysis.charts.length)
        : "";
    }
    case "board": {
      const exhibits = boardFrom(artifact).blocks.filter(
        (block) => block.type !== "prose" && block.type !== "note",
      ).length;
      return exhibits > 0 ? t("{0, plural, one {# exhibit} other {# exhibits}}", exhibits) : "";
    }
    case "portal_task": {
      const task = portalTaskFrom(artifact);
      if (task.awaitingSignInCode) return t("Needs a sign-in code");
      return task.site;
    }
    default:
      return "";
  }
}
