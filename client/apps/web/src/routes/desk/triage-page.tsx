import { TriageRoom } from "@/routes/inbox/page";

/**
 * Triage in the Desk's room: its folders, lanes and messages under the Desk's
 * own rail and strip, so a message that needs a person is worked beside the
 * rest of what needs them rather than in a page of its own.
 */
export function DeskTriagePage() {
  return <TriageRoom />;
}
