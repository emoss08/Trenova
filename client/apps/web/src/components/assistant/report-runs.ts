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
 * A tool result is a model-facing payload, not a typed contract: it is whatever
 * the tool wrote, it may be truncated, and a failed parse must leave the thread
 * rendering rather than take the message down with it.
 */
function parseRunResult(content: string): ThreadReportRun | null {
  let payload: unknown;
  try {
    payload = JSON.parse(content);
  } catch {
    return null;
  }

  if (typeof payload !== "object" || payload === null) {
    return null;
  }

  const { runId, reportKey } = payload as Record<string, unknown>;
  if (typeof runId !== "string" || runId === "") {
    return null;
  }

  return { runId, reportKey: typeof reportKey === "string" ? reportKey : "" };
}
