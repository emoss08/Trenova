import type { AgentAccess } from "@/lib/graphql/agent-access";
import type { SaveAgentDefinitionRequest } from "@/types/assistant";

/** Whether two answers to "who may use it" are the same, whatever order the roles are in. */
export function sameAccess(a: AgentAccess, b: AgentAccess): boolean {
  if (a.mode !== b.mode) {
    return false;
  }
  const left = new Set(a.roleIds);
  const right = new Set(b.roleIds);
  if (left.size !== right.size) {
    return false;
  }
  for (const id of left) {
    if (!right.has(id)) {
      return false;
    }
  }

  return true;
}

type SavedAgent = { id: string; version: number };

/**
 * Saves an edited agent and who may use it. They are two requests, so the
 * order is chosen for what a failure leaves behind.
 *
 * Access goes first. Setting it does not move the agent's version, so when
 * it fails nothing has changed and the form saves again as it stands; and
 * when the agent's own save fails after it, a retry sets the same access
 * again, which changes nothing, before saving the rest. Narrowing who may
 * use an agent also takes effect before anything else about it does.
 *
 * Access is sent only when it differs from what was last saved, and
 * `onAccessSaved` hears of it at once, so a later failure of the agent's own
 * save can say that access, at least, was saved.
 */
export async function saveEditedAgent<TAgent extends SavedAgent>({
  access,
  savedAccess,
  saveAgent,
  setAccess,
  onAccessSaved,
}: {
  access: AgentAccess;
  /** Who may use it as last saved: loaded with the agent, or set earlier in this session. */
  savedAccess: AgentAccess;
  saveAgent: () => Promise<TAgent>;
  setAccess: (access: AgentAccess) => Promise<unknown>;
  onAccessSaved?: (access: AgentAccess) => void;
}): Promise<{ agent: TAgent; accessChanged: boolean }> {
  const accessChanged = !sameAccess(access, savedAccess);
  if (accessChanged) {
    await setAccess(access);
    onAccessSaved?.(access);
  }
  const agent = await saveAgent();

  return { agent, accessChanged };
}

/**
 * What went wrong after a new agent was created. The agent exists either
 * way, so the create is not repeated; the panel says what is left to do.
 */
export type NewAgentProblem =
  /** Access could not be set, so the agent was left disabled rather than open to everyone. */
  | { kind: "access-not-saved-left-disabled"; error: unknown }
  /** Access could not be set; the agent is as it was asked to be otherwise. */
  | { kind: "access-not-saved"; error: unknown }
  /** Access was set, but the agent could not then be enabled. */
  | { kind: "not-enabled"; error: unknown };

/**
 * Creates an agent and sets who may use it.
 *
 * A new agent is created open to everyone, because the create request has
 * no access of its own. One meant only for some roles is therefore created
 * disabled, restricted, and only then enabled, so no enabled agent is ever
 * open to everyone for the moment between the two requests, and a failure
 * part-way leaves it disabled rather than open. An agent open to everyone is
 * created as asked and its roles, which matter only once it is restricted,
 * are recorded after.
 */
export async function saveNewAgent<TAgent extends SavedAgent>({
  request,
  access,
  createAgent,
  updateAgent,
  setAccess,
}: {
  request: SaveAgentDefinitionRequest;
  access: AgentAccess;
  createAgent: (request: SaveAgentDefinitionRequest) => Promise<TAgent>;
  updateAgent: (id: string, request: SaveAgentDefinitionRequest) => Promise<TAgent>;
  setAccess: (agentId: string, access: AgentAccess) => Promise<unknown>;
}): Promise<{ agent: TAgent; problem: NewAgentProblem | null }> {
  const restricted = access.mode === "Roles";
  if (!restricted && access.roleIds.length === 0) {
    return { agent: await createAgent(request), problem: null };
  }

  const holdBack = restricted && request.enabled;
  const created = await createAgent(holdBack ? { ...request, enabled: false } : request);

  try {
    await setAccess(created.id, access);
  } catch (error) {
    return {
      agent: created,
      problem: {
        kind: holdBack ? "access-not-saved-left-disabled" : "access-not-saved",
        error,
      },
    };
  }

  if (!holdBack) {
    return { agent: created, problem: null };
  }

  try {
    const enabled = await updateAgent(created.id, {
      ...request,
      enabled: true,
      version: created.version,
    });
    return { agent: enabled, problem: null };
  } catch (error) {
    return { agent: created, problem: { kind: "not-enabled", error } };
  }
}
