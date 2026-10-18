import type { PortalRunResultView } from "@/lib/portal-run";
import type { PortalTaskPayload } from "../../artifact-payloads";

/** How often an open card reads a task again while its browser is still open. */
export const PORTAL_TASK_POLL_MS = 4000;

/**
 * A portal task as the agent was told of it, for when the run itself cannot be read: what the
 * portal answered and the pages it read, with no way to act on a run this person cannot reach.
 */
export function portalTaskResultView(task: PortalTaskPayload): PortalRunResultView {
  return {
    runId: task.runId,
    status: null,
    answer: task.answer,
    failure: "",
    visitedUrls: task.pagesRead,
    offSiteUrls: [],
    hasRecording: false,
    codePrompt: task.portalSaid,
    codeExpiresAt: null,
    codeAttempts: 0,
  };
}
