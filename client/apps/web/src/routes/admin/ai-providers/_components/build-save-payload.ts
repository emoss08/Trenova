import type { AITask, SaveAIProviderRequest } from "@/types/ai-provider";

export type ProviderFormValues = Omit<SaveAIProviderRequest, "tasks"> & {
  tasks: AITask[] | null;
  preset: string;
};

/**
 * Turns form state into a save payload.
 *
 * The credential is the part that needs care. The server treats an absent
 * `apiKey` as "keep whatever is stored" and an empty string as "clear it", and
 * the edit form never receives the existing secret — so a blank field on an edit
 * has to serialize as absent, not as empty, or opening a provider and saving it
 * unchanged would wipe its credential. On a create there is nothing to preserve,
 * so blank means blank.
 */
export function buildSavePayload(
  values: ProviderFormValues,
  isEditing: boolean,
): SaveAIProviderRequest {
  const { preset: _preset, tasks, apiKey, ...rest } = values;

  const trimmedKey = apiKey?.trim() ?? "";

  return {
    ...rest,
    tasks: tasks ?? [],
    apiKey: trimmedKey !== "" ? trimmedKey : isEditing ? undefined : "",
  };
}
