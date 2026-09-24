import { assistantThreadSchema } from "@/types/assistant";
import { describe, expect, it } from "vitest";

const thread = {
  id: "athr_01JTHREAD00000000000000",
  businessUnitId: "bu_01JBU0000000000000000000",
  organizationId: "org_01JORG000000000000000000",
  userId: "usr_01JUSER00000000000000000",
  agentDefinitionId: "agdef_01JAGENT000000000000000",
  status: "Active",
  createdAt: 1_758_000_000,
  updatedAt: 1_758_000_000,
};

/**
 * A conversation that cannot continue says why, so the notice in place of the
 * composer can tell a switched-off agent from one the reader lost.
 */
describe("assistantThreadSchema cannotContinueReason", () => {
  it("reads each reason the server gives", () => {
    for (const reason of ["AgentDeleted", "AgentDisabled", "AgentNotConversational", "NoAccess"]) {
      const parsed = assistantThreadSchema.parse({
        ...thread,
        canContinue: false,
        cannotContinueReason: reason,
      });

      expect(parsed.cannotContinueReason).toBe(reason);
    }
  });

  it("has no reason for a conversation that can continue", () => {
    const parsed = assistantThreadSchema.parse({ ...thread, canContinue: true });

    expect(parsed.canContinue).toBe(true);
    expect(parsed.cannotContinueReason).toBeUndefined();
  });

  // A reason a newer server adds must not stop the conversation opening; it
  // reads as no reason, and the notice falls back to its plain words.
  it("reads a reason this build does not know as none", () => {
    const parsed = assistantThreadSchema.parse({
      ...thread,
      canContinue: false,
      cannotContinueReason: "AgentArchived",
    });

    expect(parsed.canContinue).toBe(false);
    expect(parsed.cannotContinueReason).toBeUndefined();
  });
});
