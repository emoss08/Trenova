import { isTypingTarget } from "@/lib/dom";

type ApproveKeyEvent = Pick<KeyboardEvent, "key" | "metaKey" | "ctrlKey" | "defaultPrevented"> & {
  target: EventTarget | null;
};

/**
 * Whether Cmd/Ctrl+Enter, pressed in the Desk's stage, approves the change waiting on the
 * person. A key something below already handled (the message box sending with Cmd+Enter) never
 * approves, and neither does one pressed in a field that holds text: a person typing a
 * question about a change is asking, not deciding. An empty message box, the card itself or
 * anywhere else in the stage approves.
 */
export function approvesFromKey(event: ApproveKeyEvent): boolean {
  if (event.defaultPrevented) return false;
  if (event.key !== "Enter" || !(event.metaKey || event.ctrlKey)) return false;
  return !holdsText(event.target);
}

function holdsText(target: EventTarget | null): boolean {
  if (!isTypingTarget(target)) return false;
  if (target instanceof HTMLTextAreaElement || target instanceof HTMLInputElement) {
    return target.value.trim() !== "";
  }
  return (target as HTMLElement).textContent?.trim() !== "";
}
