import { cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { AssistantAgentProvider, useDelegateIdentity } from "../agent-context";
import type { AgentIdentityInput } from "../agent-identity";

afterEach(cleanup);

const LISTED: AgentIdentityInput = {
  id: "agdef_rb",
  name: "Report Builder (listed)",
  icon: "receipt",
  accent: "amber",
  template: "BillingAssistant",
};

function identityOf(own: AgentIdentityInput, delegates: AgentIdentityInput[] = [LISTED]) {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <AssistantAgentProvider agent={null} delegates={delegates}>
      {children}
    </AssistantAgentProvider>
  );

  return renderHook(() => useDelegateIdentity(own), { wrapper }).result.current;
}

describe("useDelegateIdentity", () => {
  it("draws the mark the hand-off carries before the thread's list of delegates", () => {
    expect(
      identityOf({ id: "agdef_rb", name: "Report Builder", icon: "truck", accent: "teal" }),
    ).toMatchObject({ name: "Report Builder", icon: "truck", accent: "teal" });
  });

  it("fills what the hand-off does not carry from the list", () => {
    expect(identityOf({ id: "agdef_rb", name: "Report Builder", icon: "", accent: "" })).toEqual({
      id: "agdef_rb",
      name: "Report Builder",
      icon: "receipt",
      accent: "amber",
      template: "BillingAssistant",
    });
  });

  it("keeps the hand-off's own identity for an agent the list does not hold", () => {
    const own = { id: "agdef_gone", name: "Gone Agent", icon: "", accent: "" };

    expect(identityOf(own)).toEqual(own);
  });
});
