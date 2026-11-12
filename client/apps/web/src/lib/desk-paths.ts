/** Needs you narrowed to what agents ask to change: the one place changes are decided. */
export const DESK_DECISIONS_PATH = "/desk/needs?lens=decisions";

/** Decisions opened on one change, or on the change one agent's run is waiting on. */
export function deskDecisionPath(target: { proposal: string } | { run: string }): string {
  const [name, value] = "proposal" in target ? ["proposal", target.proposal] : ["run", target.run];
  return `${DESK_DECISIONS_PATH}&${name}=${encodeURIComponent(value)}`;
}
