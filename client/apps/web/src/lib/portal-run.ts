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
    AwaitingCode: {
      phase: "awaiting",
      text: t("Needs a code"),
      description: t(
        "The portal sent a sign-in code to the person the task is for, and the browser is waiting for it",
      ),
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

const CODE_MARKER = /^\s*SIGN-IN CODE NEEDED[\s:.,-]*/iu;

/**
 * How the portal asked for its sign-in code, as the browser put it, without the marker the
 * browser was told to begin with.
 */
export function portalCodePrompt(prompt: string): string {
  return prompt.replace(CODE_MARKER, "").trim();
}

/** A task whose browser is still open: running, or waiting on a sign-in code. */
export function isLivePortalRun(status: PortalRunStatus | null): boolean {
  return status === "Running" || status === "AwaitingCode";
}

/** What a person reads of a portal task's outcome, from the run or from what the agent was told. */
export type PortalRunResultView = {
  runId: string;
  status: PortalRunStatus | null;
  answer: string;
  failure: string;
  visitedUrls: string[];
  offSiteUrls: string[];
  hasRecording: boolean;
  codePrompt: string;
  codeExpiresAt: number | null;
  codeAttempts: number;
};

export function portalRunResultView(run: PortalRun): PortalRunResultView {
  return {
    runId: run.id,
    status: run.status,
    answer: run.answer,
    failure: run.failure,
    visitedUrls: run.visitedUrls,
    offSiteUrls: run.offSiteUrls,
    hasRecording: run.hasRecording,
    codePrompt: run.codePrompt,
    codeExpiresAt: run.codeExpiresAt ?? null,
    codeAttempts: run.codeAttempts,
  };
}
