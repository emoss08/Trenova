import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import type { PortalRun, PortalRunStatus } from "@/types/agent-extension";

/** Where a portal task stands: running, done, or how it ended without succeeding. */
export function portalRunStatusAttrs(t: TranslateFn): Record<PortalRunStatus, BadgeAttrProps> {
  return {
    Running: {
      phase: "active",
      text: t("Running"),
      description: t("The browser is working on it"),
    },
    Succeeded: {
      phase: "complete",
      text: t("Done"),
      description: t("The portal did what was asked and said so"),
    },
    Failed: {
      phase: "failed",
      text: t("Failed"),
      description: t("The browser could not finish the task"),
    },
    Stopped: {
      phase: "closed",
      text: t("Stopped"),
      description: t("The task was stopped before it finished"),
    },
    TimedOut: {
      phase: "failed",
      text: t("Timed out"),
      description: t("The task ran out of time and was stopped"),
    },
    Strayed: {
      phase: "attention",
      text: t("Left the site"),
      description: t(
        "The browser reached pages outside the site's addresses, so what it found was withheld",
      ),
    },
  };
}

/** What the task did and where: looked something up or acted, on a site that may be gone. */
export function portalRunTitle(t: TranslateFn, run: Pick<PortalRun, "mode" | "siteName">): string {
  if (run.mode === "Act") {
    return run.siteName ? t("Acted on {0}", run.siteName) : t("Acted on a removed site");
  }
  return run.siteName ? t("Looked up on {0}", run.siteName) : t("Looked up on a removed site");
}
