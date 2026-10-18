import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import type { WebWatchCadence, WebWatchStatus } from "@/types/agent-extension";

/** Where a web watch stands: still looking, or how it ended. */
export function webWatchStatusAttrs(t: TranslateFn): Record<WebWatchStatus, BadgeAttrProps> {
  return {
    Active: {
      phase: "active",
      text: t("Watching"),
      description: t("The vendor looks on schedule and brings back only what is new"),
    },
    Ended: {
      phase: "closed",
      text: t("Ended"),
      description: t("The watch's time ran out"),
    },
    Stopped: {
      phase: "closed",
      text: t("Stopped"),
      description: t("A person or an agent stopped the watch"),
    },
    Failed: {
      phase: "failed",
      text: t("Failed"),
      description: t("The vendor could no longer run the watch"),
    },
  };
}

/** How often a watch looks, as a person reads it. */
export function webWatchCadenceLabel(t: TranslateFn, cadence: WebWatchCadence | string): string {
  switch (cadence) {
    case "SixHours":
      return t("Every six hours");
    case "Weekly":
      return t("Weekly");
    default:
      return t("Daily");
  }
}
