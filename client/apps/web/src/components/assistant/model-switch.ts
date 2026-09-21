import type { AssistantProviderOption } from "@/types/assistant";

export type ModelSwitchNotice = {
  /** The model the conversation is moving to; null for the organization's order. */
  to: string | null;
  /** The model it was on; null for the organization's order. */
  from: string | null;
};

/**
 * Whether the picker's choice differs from the model the conversation has
 * been on, and what to call both.
 *
 * A new model does not remember the conversation: it reads the whole
 * thread again before it answers, which takes longer and can cost more.
 * That is worth a line at the moment of switching, and nothing when no
 * reply exists yet to be re-read.
 */
export function modelSwitchNotice({
  pickedId,
  savedId,
  hasReplies,
  providers,
}: {
  pickedId: string;
  savedId: string;
  hasReplies: boolean;
  providers: readonly AssistantProviderOption[];
}): ModelSwitchNotice | null {
  if (!hasReplies || pickedId === savedId) {
    return null;
  }
  const nameOf = (id: string) =>
    id === "" ? null : (providers.find((p) => p.id === id)?.name ?? id);

  return { to: nameOf(pickedId), from: nameOf(savedId) };
}
