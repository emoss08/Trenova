import { cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
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

  // The thread publishes its agent's delegates as `myAgents` served them, so
  // an old hand-off saved without a mark still draws the delegate's.
  it("draws an old hand-off from the delegates the thread's agent was served with", () => {
    const thread: AgentChoice = {
      id: "agdef_widgets",
      name: "Homepage Widget Builder",
      description: "",
      template: null,
      icon: "",
      accent: "",
      toolNames: [],
      systemKey: "",
      starters: [],
      delegates: [
        { id: "agdef_rb", name: "Report Builder", icon: "receipt", accent: "teal", template: null },
      ],
    };
    const wrapper = ({ children }: { children: ReactNode }) => (
      <AssistantAgentProvider agent={thread} delegates={thread.delegates}>
        {children}
      </AssistantAgentProvider>
    );

    const drawn = renderHook(
      () => useDelegateIdentity({ id: "agdef_rb", name: "Report Builder", icon: "", accent: "" }),
      { wrapper },
    ).result.current;

    expect(drawn).toMatchObject({ icon: "receipt", accent: "teal" });
  });

  it("keeps the hand-off's own identity for an agent the list does not hold", () => {
    const own = { id: "agdef_gone", name: "Gone Agent", icon: "", accent: "" };

    expect(identityOf(own)).toEqual(own);
  });
});
