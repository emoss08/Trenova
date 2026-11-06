/** The DOM id of a queue row, so the keyboard can move focus to the move it selects. */
export function queueRowId(moveId: string): string {
  return `dispatch-queue-${moveId}`;
}
