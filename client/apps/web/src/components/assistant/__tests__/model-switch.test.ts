import { describe, expect, it } from "vitest";
import { modelSwitchNotice } from "../model-switch";

const providers = [
  { id: "aiprv_a", name: "Claude", kind: "AnthropicMessages", model: "claude", trusted: true },
  { id: "aiprv_b", name: "Router", kind: "OpenAIChat", model: "openrouter/free", trusted: false },
];

/**
 * Switching models mid-conversation means the new model reads the whole
 * thread again before it answers, which takes longer and can cost more.
 * The reader is told at the moment they switch, not after the reply.
 */
describe("modelSwitchNotice", () => {
  it("names the model the conversation is moving to", () => {
    expect(
      modelSwitchNotice({ pickedId: "aiprv_b", savedId: "aiprv_a", hasReplies: true, providers }),
    ).toEqual({ to: "Router", from: "Claude" });
  });

  it("says so for a move to automatic and from automatic", () => {
    expect(
      modelSwitchNotice({ pickedId: "", savedId: "aiprv_a", hasReplies: true, providers }),
    ).toEqual({ to: null, from: "Claude" });
    expect(
      modelSwitchNotice({ pickedId: "aiprv_a", savedId: "", hasReplies: true, providers }),
    ).toEqual({ to: "Claude", from: null });
  });

  it("is silent when nothing changed or nothing has been said yet", () => {
    expect(
      modelSwitchNotice({ pickedId: "aiprv_a", savedId: "aiprv_a", hasReplies: true, providers }),
    ).toBeNull();
    expect(
      modelSwitchNotice({ pickedId: "aiprv_b", savedId: "aiprv_a", hasReplies: false, providers }),
    ).toBeNull();
  });
});
