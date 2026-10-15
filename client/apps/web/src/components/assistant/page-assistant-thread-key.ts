// Kept apart from page-assistant.tsx: a page that only resets its conversation imports this
// key, and importing anything from the component module brings the Desk thread with it.

/** The cache key of a page's opened conversation, for a page that must start it afresh. */
export function pageAssistantThreadKey(conversationKey: readonly unknown[]): readonly unknown[] {
  return ["page-assistant-thread", ...conversationKey];
}
