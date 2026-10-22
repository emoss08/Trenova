import { describe, expect, it } from "vitest";
import {
  advisorBlock,
  refusalInForce,
} from "@/routes/agent-control/_components/providers/agent-settings-model";

/**
 * The refusal the server records (aiprovider.AdvisorRefusal) names the advisor
 * the refused call declared and the provider's model; it is cleared when the
 * advisor, the model or the platform changes.
 */
const refusal = {
  advisor: "claude-opus-5-5",
  model: "claude-haiku-4-5",
  message: "claude-opus-5-5 cannot advise claude-haiku-4-5",
  refusedAt: 1_792_000_000,
};

describe("refusalInForce", () => {
  it("shows the refusal while the draft pairs the default advisor with the refused model", () => {
    expect(
      refusalInForce(refusal, {
        advisorEnabled: true,
        advisorModel: "",
        model: "claude-haiku-4-5",
      }),
    ).toBe(refusal);
  });

  it("shows the refusal for an advisor named outright", () => {
    expect(
      refusalInForce(refusal, {
        advisorEnabled: true,
        advisorModel: " claude-opus-5-5 ",
        model: "claude-haiku-4-5",
      }),
    ).toBe(refusal);
  });

  it("drops the refusal once the draft names another advisor", () => {
    expect(
      refusalInForce(refusal, {
        advisorEnabled: true,
        advisorModel: "claude-opus-5-6",
        model: "claude-haiku-4-5",
      }),
    ).toBeNull();
  });

  it("drops the refusal once the draft runs another model", () => {
    expect(
      refusalInForce(refusal, {
        advisorEnabled: true,
        advisorModel: "",
        model: "claude-sonnet-5",
      }),
    ).toBeNull();
  });

  it("shows nothing while the advisor is off", () => {
    expect(
      refusalInForce(refusal, {
        advisorEnabled: false,
        advisorModel: "",
        model: "claude-haiku-4-5",
      }),
    ).toBeNull();
  });

  it("shows nothing when nothing was refused", () => {
    expect(
      refusalInForce(null, { advisorEnabled: true, advisorModel: "", model: "claude-haiku-4-5" }),
    ).toBeNull();
  });
});

describe("advisorBlock", () => {
  it("lets a Claude API chat provider consult the advisor", () => {
    expect(
      advisorBlock({ kind: "AnthropicMessages", platform: "Direct", tasks: ["AssistantChat"] }),
    ).toBeNull();
  });

  it("lets a Claude Platform on AWS chat provider consult the advisor", () => {
    expect(
      advisorBlock({
        kind: "AnthropicMessages",
        platform: "ClaudePlatformAWS",
        tasks: ["AssistantChat", "General"],
      }),
    ).toBeNull();
  });

  it("names the protocol first for a provider that does not speak Anthropic's", () => {
    expect(advisorBlock({ kind: "OpenAIChat", platform: "Direct", tasks: [] })).toBe("kind");
  });

  it.each(["Bedrock", "VertexAI", "ClaudePlatformGoogleCloud", "Foundry"] as const)(
    "names the platform for %s",
    (platform) => {
      expect(advisorBlock({ kind: "AnthropicMessages", platform, tasks: ["AssistantChat"] })).toBe(
        "platform",
      );
    },
  );

  it("names the task for a provider that does not handle chat", () => {
    expect(advisorBlock({ kind: "AnthropicMessages", platform: "Direct", tasks: null })).toBe(
      "task",
    );
  });
});
