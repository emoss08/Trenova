import {
  ADVISOR_PLATFORMS,
  DEFAULT_ADVISOR_MODEL,
  type AIProviderAdvisorRefusal,
  type AIProviderKind,
  type AIProviderPlatform,
  type AITask,
} from "@/types/ai-provider";

/** The task whose calls declare the advisor and may answer with a script. */
export const CHAT_TASK: AITask = "AssistantChat";

/**
 * Why a provider cannot consult Anthropic's advisor: the protocol is not
 * Anthropic's, the platform does not serve the tool, or the provider does not
 * handle chat. Null when it can.
 */
export type AdvisorBlock = "kind" | "platform" | "task";

export function advisorBlock(draft: {
  kind: AIProviderKind;
  platform: AIProviderPlatform;
  tasks: readonly AITask[] | null;
}): AdvisorBlock | null {
  if (draft.kind !== "AnthropicMessages") return "kind";
  if (!(ADVISOR_PLATFORMS as readonly AIProviderPlatform[]).includes(draft.platform)) {
    return "platform";
  }
  if (!(draft.tasks ?? []).includes(CHAT_TASK)) return "task";
  return null;
}

/** Whether a provider's model may answer with a script: only one that handles chat. */
export function scriptsAllowed(draft: { tasks: readonly AITask[] | null }): boolean {
  return (draft.tasks ?? []).includes(CHAT_TASK);
}

/**
 * The advisor and script settings a draft saves. A setting the provider cannot
 * take is saved off, so the server never refuses a draft for a switch the
 * editor no longer shows, and script writes are never on without scripts.
 */
export function agentSettingsPayload(draft: {
  kind: AIProviderKind;
  platform: AIProviderPlatform;
  tasks: readonly AITask[] | null;
  advisorEnabled: boolean;
  advisorModel: string;
  scripts: boolean;
  scriptWrites: boolean;
}): {
  advisorEnabled: boolean;
  advisorModel: string;
  scripts: boolean;
  scriptWrites: boolean;
} {
  const scripts = scriptsAllowed(draft) && draft.scripts;
  return {
    advisorEnabled: advisorBlock(draft) === null && draft.advisorEnabled,
    advisorModel: draft.kind === "AnthropicMessages" ? draft.advisorModel.trim() : "",
    scripts,
    scriptWrites: scripts && draft.scriptWrites,
  };
}

/**
 * The API's refusal of the advisor, while the draft still pairs that advisor
 * with that model. The server clears a refusal once either changes, so one for
 * another pairing than the draft's no longer holds, and nothing is refused
 * while the advisor is off.
 */
export function refusalInForce(
  refusal: AIProviderAdvisorRefusal | null,
  draft: { advisorEnabled: boolean; advisorModel: string; model: string },
): AIProviderAdvisorRefusal | null {
  if (refusal === null || !draft.advisorEnabled) return null;
  const advisor = draft.advisorModel.trim() || DEFAULT_ADVISOR_MODEL;
  return refusal.advisor === advisor && refusal.model === draft.model.trim() ? refusal : null;
}
