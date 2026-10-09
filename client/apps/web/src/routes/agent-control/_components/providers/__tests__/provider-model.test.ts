import { describe, expect, it } from "vitest";
import type { AIProviderKind } from "@/types/ai-provider";
import {
  baseUrlProblem,
  firstsOf,
  formatLatency,
  formatContext,
  formatTokens,
  isPrivateAddress,
  keyField,
  liveState,
  moveBy,
  moveTo,
  routeOf,
  skipReason,
  taskMetas,
  toggleTask,
  uncovered,
  type RoutingProvider,
  type TaskMeta,
} from "../provider-model";

const keyRequired = (kind: AIProviderKind) =>
  kind === "AnthropicMessages" || kind === "OpenAIResponses";

function provider(overrides: Partial<RoutingProvider> & { id: string }): RoutingProvider {
  return {
    name: overrides.id,
    kind: "Ollama",
    tasks: [],
    trusted: false,
    enabled: true,
    hasApiKey: false,
    embeddingDimensions: null,
    ...overrides,
  };
}

const chat: TaskMeta = { task: "AssistantChat", label: "Assistant chat", trust: false };
const billing: TaskMeta = { task: "BillingDiagnosis", label: "Billing diagnosis", trust: true };
const embedding: TaskMeta = { task: "Embedding", label: "Embedding", trust: false };

/**
 * The rule is the server's Provider.CanServeTask: enabled, assigned, trusted for a task
 * that writes to the ledger, and for Embedding a protocol with an embedding endpoint and
 * a supported vector size. The grid must never show a route the router would refuse.
 */
describe("routeOf", () => {
  it("takes the first enabled, assigned provider in order and names the next", () => {
    const providers = [
      provider({ id: "a", tasks: ["General"] }),
      provider({ id: "b", tasks: ["AssistantChat"] }),
      provider({ id: "c", tasks: ["AssistantChat"] }),
    ];
    const route = routeOf(chat, providers, keyRequired);
    expect(route.first?.id).toBe("b");
    expect(route.next?.id).toBe("c");
    expect(route.assigned.map((entry) => entry.id)).toEqual(["b", "c"]);
  });

  it("passes over a provider that is off, even when it is assigned first", () => {
    const providers = [
      provider({ id: "a", tasks: ["AssistantChat"], enabled: false }),
      provider({ id: "b", tasks: ["AssistantChat"] }),
    ];
    expect(routeOf(chat, providers, keyRequired).first?.id).toBe("b");
  });

  it("refuses an untrusted provider a task that writes to the ledger", () => {
    const providers = [
      provider({ id: "a", tasks: ["BillingDiagnosis"] }),
      provider({ id: "b", tasks: ["BillingDiagnosis"], trusted: true }),
    ];
    expect(routeOf(billing, providers, keyRequired).first?.id).toBe("b");
    expect(skipReason(providers[0]!, billing, keyRequired)).toBe("untrusted");
  });

  it("refuses Embedding to a protocol without an endpoint or without a vector size", () => {
    const anthropic = provider({
      id: "a",
      kind: "AnthropicMessages",
      tasks: ["Embedding"],
      hasApiKey: true,
      embeddingDimensions: 1024,
    });
    const sizeless = provider({ id: "b", tasks: ["Embedding"] });
    const sized = provider({ id: "c", tasks: ["Embedding"], embeddingDimensions: 768 });
    expect(skipReason(anthropic, embedding, keyRequired)).toBe("noEmbedding");
    expect(skipReason(sizeless, embedding, keyRequired)).toBe("noEmbedding");
    expect(routeOf(embedding, [anthropic, sizeless, sized], keyRequired).first?.id).toBe("c");
  });

  it("says a disabled provider without the key its protocol needs is waiting for a key", () => {
    const keyless = provider({
      id: "a",
      kind: "AnthropicMessages",
      tasks: ["AssistantChat"],
      enabled: false,
    });
    expect(skipReason(keyless, chat, keyRequired)).toBe("needsKey");
    expect(skipReason({ ...keyless, hasApiKey: true }, chat, keyRequired)).toBe("off");
  });

  it("counts what a provider takes first and what nobody takes", () => {
    const providers = [
      provider({ id: "a", tasks: ["AssistantChat", "Embedding"] }),
      provider({ id: "b", tasks: ["AssistantChat", "BillingDiagnosis"] }),
    ];
    const metas = [chat, billing, embedding];
    expect(firstsOf(providers[0]!, metas, providers, keyRequired).map((m) => m.task)).toEqual([
      "AssistantChat",
    ]);
    expect(firstsOf(providers[1]!, metas, providers, keyRequired)).toEqual([]);
    expect(uncovered(metas, providers, keyRequired).map((m) => m.task)).toEqual([
      "BillingDiagnosis",
      "Embedding",
    ]);
  });
});

describe("taskMetas", () => {
  it("lists the catalog's tasks in grid order with General last and drops unknown ones", () => {
    const metas = taskMetas([
      {
        task: "General",
        label: "General",
        description: "",
        requiresTrust: false,
        volumeGuidance: "",
      },
      {
        task: "BillingDiagnosis",
        label: "Billing diagnosis",
        description: "",
        requiresTrust: true,
        volumeGuidance: "",
      },
      {
        task: "AssistantChat",
        label: "Assistant chat",
        description: "",
        requiresTrust: false,
        volumeGuidance: "",
      },
    ]);
    expect(metas.map((meta) => meta.task)).toEqual([
      "AssistantChat",
      "BillingDiagnosis",
      "General",
    ]);
    expect(metas[1]!.trust).toBe(true);
  });
});

describe("ordering", () => {
  it("moves one step and stays put at either end", () => {
    expect(moveBy(["a", "b", "c"], "b", -1)).toEqual(["b", "a", "c"]);
    expect(moveBy(["a", "b", "c"], "b", 1)).toEqual(["a", "c", "b"]);
    expect(moveBy(["a", "b", "c"], "a", -1)).toEqual(["a", "b", "c"]);
    expect(moveBy(["a", "b", "c"], "c", 1)).toEqual(["a", "b", "c"]);
  });

  it("drops a dragged provider where another stands, either way", () => {
    expect(moveTo(["a", "b", "c", "d"], "a", "c")).toEqual(["b", "c", "a", "d"]);
    expect(moveTo(["a", "b", "c", "d"], "d", "b")).toEqual(["a", "d", "b", "c"]);
    expect(moveTo(["a", "b"], "a", "missing")).toEqual(["a", "b"]);
  });

  it("toggles a task without reordering the rest", () => {
    expect(toggleTask(["General", "AssistantChat"], "General")).toEqual(["AssistantChat"]);
    expect(toggleTask(["General"], "Embedding")).toEqual(["General", "Embedding"]);
  });
});

describe("base URL", () => {
  it("knows a private address from a public one", () => {
    for (const url of [
      "http://localhost:11434",
      "http://127.0.0.1:8000/v1",
      "http://10.0.0.5:8000",
      "http://172.20.1.1",
      "http://192.168.1.9:1234",
      "http://gpu-box:8000",
      "http://gpu.local:8000",
      "http://[::1]:8000",
    ]) {
      expect(isPrivateAddress(url), url).toBe(true);
    }
    for (const url of [
      "https://api.groq.com/openai/v1",
      "http://172.32.0.1",
      "https://api.openai.com",
    ]) {
      expect(isPrivateAddress(url), url).toBe(false);
    }
  });

  it("asks for a scheme, then for Private network on a private address", () => {
    expect(baseUrlProblem("", false)).toBeNull();
    expect(baseUrlProblem("api.example.com/v1", false)).toBe("scheme");
    expect(baseUrlProblem("http://localhost:8000/v1", false)).toBe("private");
    expect(baseUrlProblem("http://localhost:8000/v1", true)).toBeNull();
    expect(baseUrlProblem("https://api.example.com/v1", false)).toBeNull();
  });
});

describe("liveState and latency", () => {
  const ok = {
    success: true,
    message: "Connected",
    modelIdentifier: "",
    schemaHonoured: true,
    latencyMs: 200,
    detail: "",
    testedAt: 1,
  };

  it("puts a test in flight first, then a failure since the last test, then the last test", () => {
    expect(liveState(ok, true, true)).toBe("run");
    expect(liveState(ok, false, true)).toBe("fail");
    expect(liveState(ok, false, false)).toBe("ok");
    expect(liveState({ ...ok, success: false }, false, false)).toBe("fail");
    expect(liveState(null, false, false)).toBe("none");
  });

  it("writes milliseconds under a second and seconds above", () => {
    expect(formatLatency(410)).toBe("410ms");
    expect(formatLatency(1420)).toBe("1.4s");
  });
});

describe("formatTokens", () => {
  it("writes tokens the way the read sheet does", () => {
    expect(formatTokens(820)).toBe("820");
    expect(formatTokens(14_200)).toBe("14k");
    expect(formatTokens(1_400)).toBe("1.4k");
    expect(formatTokens(900_000)).toBe("0.9M");
    expect(formatTokens(1_400_000)).toBe("1.4M");
  });
});

describe("formatContext", () => {
  it("writes a context window the way model cards do", () => {
    expect(formatContext(32_768)).toBe("32k");
    expect(formatContext(131_072)).toBe("128k");
    expect(formatContext(200_000)).toBe("200k");
    expect(formatContext(1_000_000)).toBe("1M");
    expect(formatContext(1_048_576)).toBe("1M");
    expect(formatContext(2_500_000)).toBe("2.5M");
  });
});

describe("keyField", () => {
  it("offers a key for a protocol or preset that needs one, a stored key, or a hosted address", () => {
    expect(keyField({ mandatory: true, hasStoredKey: false, baseUrl: "" })).toBe(true);
    expect(
      keyField({ mandatory: false, hasStoredKey: true, baseUrl: "http://localhost:8000" }),
    ).toBe(true);
    expect(
      keyField({
        mandatory: false,
        hasStoredKey: false,
        baseUrl: "https://api.groq.com/openai/v1",
      }),
    ).toBe(true);
    expect(
      keyField({ mandatory: false, hasStoredKey: false, baseUrl: "http://localhost:11434" }),
    ).toBe(false);
    expect(keyField({ mandatory: false, hasStoredKey: false, baseUrl: "" })).toBe(false);
  });
});
