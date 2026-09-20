import { parseToolResult } from "./tool-presentation";
import type { ToolExchange } from "./thread-view";

/** A report run the assistant started, as the thread records it. */
export type ThreadReportRun = {
  runId: string;
  reportKey: string;
};

/** The tools whose results name a run worth following. */
const RUN_BEARING_TOOLS = new Set(["run_report", "get_report_run"]);

/**
 * Pulls the report runs out of a turn's tool traffic.
 *
 * A run started in a conversation used to leave nothing behind but a sentence
 * with an id in it, so the only way to learn the outcome was to ask again. The
 * id is in the tool result all along; reading it back is what lets the thread
 * follow the run itself.
 *
 * Deduplicated by run id and kept in the order the thread mentions them, since
 * a turn that starts a run and a later turn that checks on it name the same run
 * and should not stack up two cards for it.
 */
export function reportRunsFrom(tools: readonly ToolExchange[]): ThreadReportRun[] {
  const runs = new Map<string, ThreadReportRun>();

  for (const exchange of tools) {
    if (!RUN_BEARING_TOOLS.has(exchange.call.name)) {
      continue;
    }
    const result = exchange.result;
    if (result === null || result.toolFailed || result.content === "") {
      continue;
    }

    const parsed = parseRunResult(result.content);
    if (parsed !== null && !runs.has(parsed.runId)) {
      runs.set(parsed.runId, parsed);
    }
  }

  return [...runs.values()];
}

/**
 * A saved tool result is not the JSON the tool returned. The runtime fences it
 * as untrusted data under a "Result from <tool>" line and may cut a long one
 * short, so JSON.parse on the stored content fails on every single result —
 * which is exactly how this shipped showing no card at all. parseToolResult is
 * what already knows that shape; going through it is both the fix and the only
 * way this stays correct when the fence changes.
 *
 * A result that is truncated, a failure, or not JSON yields nothing rather than
 * taking the message down with it.
 */
function parseRunResult(content: string): ThreadReportRun | null {
  const result = parseToolResult(content);
  if (result.kind !== "json") {
    return null;
  }

  const payload = result.value;
  if (typeof payload !== "object" || payload === null) {
    return null;
  }

  const { runId, reportKey } = payload as Record<string, unknown>;
  if (typeof runId !== "string" || runId === "") {
    return null;
  }

  return { runId, reportKey: typeof reportKey === "string" ? reportKey : "" };
}
