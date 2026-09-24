import type { AgentAccess } from "@/lib/graphql/agent-access";

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

/** Who may use an agent the server creates without being told: everyone, with no roles. */
export const NEW_AGENT_ACCESS: AgentAccess = { mode: "Everyone", roleIds: [] };

/**
 * Who may use the agent, to send with its save, or undefined to leave it as
 * the server has it.
 *
 * The save sets it in the same transaction as the rest of the agent, so an
 * agent meant for some roles is created restricted and enabled in one request,
 * never open to everyone for a moment and never left half-saved. It is sent
 * only when it differs from what was last saved: changing it needs permission
 * to update roles, which a person editing only the agent's instructions may
 * not hold, and re-sending it unchanged would put back roles someone else
 * changed in the meantime.
 */
export function accessToSave(access: AgentAccess, saved: AgentAccess): AgentAccess | undefined {
  return sameAccess(access, saved) ? undefined : access;
}
