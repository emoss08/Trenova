import type { AITask, SaveAIProviderRequest } from "@/types/ai-provider";

export type ProviderFormValues = Omit<
  SaveAIProviderRequest,
  "tasks" | "extraBody" | "embeddingDimensions"
> & {
  tasks: AITask[] | null;
  preset: string;
  /** The vendor fields as JSON text; the form edits text, the server takes an object. */
  extraBodyText: string;
  /**
   * The vector size as the select holds it: a string, blank for none. The
   * server takes a number or null.
   */
  embeddingDimensionsChoice: string;
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
  const {
    preset: _preset,
    tasks,
    apiKey,
    extraBodyText,
    embeddingDimensionsChoice,
    ...rest
  } = values;

  const trimmedKey = apiKey?.trim() ?? "";
  const assigned = tasks ?? [];
  // Only a provider that embeds carries a vector size or an input style; one
  // that does not is saved with neither, so a later retask starts clean.
  const embeds = assigned.includes("Embedding");

  return {
    ...rest,
    tasks: assigned,
    apiKey: trimmedKey !== "" ? trimmedKey : isEditing ? undefined : "",
    extraBody: parseExtraBody(extraBodyText),
    embeddingDimensions: embeds ? parseEmbeddingDimensions(embeddingDimensionsChoice) : null,
    embeddingInputStyle: embeds ? rest.embeddingInputStyle : "None",
  };
}

/**
 * Reads the chosen vector size. Blank is none; anything else went through the
 * form schema, which only admits the sizes the server accepts.
 */
export function parseEmbeddingDimensions(choice: string): number | null {
  const trimmed = choice.trim();
  if (trimmed === "") {
    return null;
  }
  const parsed = Number(trimmed);
  return Number.isInteger(parsed) ? parsed : null;
}

/**
 * Reads the vendor fields the form edited as text. The schema has already
 * refused anything that is not a JSON object, so a parse failure here can
 * only mean the value never went through validation; null is the safe
 * reading of that, since it sends no vendor fields rather than a guess.
 */
function parseExtraBody(text: string): Record<string, unknown> | null {
  const trimmed = text.trim();
  if (trimmed === "") {
    return null;
  }

  try {
    const parsed: unknown = JSON.parse(trimmed);
    if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
      return null;
    }
    return Object.keys(parsed).length > 0 ? (parsed as Record<string, unknown>) : null;
  } catch {
    return null;
  }
}
