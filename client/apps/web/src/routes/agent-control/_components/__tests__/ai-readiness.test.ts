import type { AIProvider, AITaskDescriptor } from "@/types/ai-provider";
import { describe, expect, it } from "vitest";
import { assessReadiness } from "../ai-readiness";

function provider(overrides: Partial<AIProvider> = {}): AIProvider {
  return {
    id: "aiprv_1",
    businessUnitId: "bu_1",
    organizationId: "org_1",
    name: "Local Ollama",
    description: "",
    kind: "Ollama",
    baseUrl: "",
    model: "qwen2.5:14b",
    allowPrivateNetwork: true,
    structuredOutputMode: "JSONSchema",
    maxTokens: 8192,
    tasks: ["AssistantChat"],
    priority: 10,
    trusted: false,
    enabled: true,
    version: 0,
    createdAt: 0,
    updatedAt: 0,
    ...overrides,
  };
}

function task(overrides: Partial<AITaskDescriptor> = {}): AITaskDescriptor {
  return {
    task: "AssistantChat",
    label: "Assistant chat",
    description: "",
    requiresTrust: false,
    volumeGuidance: "",
    ...overrides,
  };
}

/**
 * The readiness banner answers one question for an administrator: if I turn an
 * agent on, will it work? That depends on which tasks have a provider that can
 * actually serve them right now, which is the same rule the server's router
 * applies — enabled, assigned, and trusted where the task needs it.
 */
describe("assessReadiness", () => {
  it("reports a task covered when an enabled provider serves it", () => {
    const readiness = assessReadiness([provider()], [task()]);

    expect(readiness.uncovered).toEqual([]);
    expect(readiness.assistantReady).toBe(true);
  });

  // A disabled provider is not a candidate. The form lets one be saved that way
  // on purpose (a hosted model waiting for a key), and it must not count.
  it("does not count a disabled provider", () => {
    const readiness = assessReadiness([provider({ enabled: false })], [task()]);

    expect(readiness.uncovered.map((entry) => entry.task)).toEqual(["AssistantChat"]);
    expect(readiness.assistantReady).toBe(false);
  });

  // The server refuses an untrusted provider for a task that reaches the
  // ledger. Reporting it as covered would tell an administrator the billing
  // agent is ready when its first run would fail.
  it("does not count an untrusted provider for a task that requires trust", () => {
    const billing = task({
      task: "BillingDiagnosis",
      label: "Billing diagnosis",
      requiresTrust: true,
    });
    const untrusted = provider({ tasks: ["BillingDiagnosis"], trusted: false });
    const trusted = provider({ id: "aiprv_2", tasks: ["BillingDiagnosis"], trusted: true });

    expect(assessReadiness([untrusted], [billing]).uncovered.map((e) => e.task)).toEqual([
      "BillingDiagnosis",
    ]);
    expect(assessReadiness([untrusted, trusted], [billing]).uncovered).toEqual([]);
  });

  it("lists every uncovered task in the order the catalog gives them", () => {
    const tasks = [
      task({ task: "ScopeClassification", label: "Scope classification" }),
      task(),
      task({ task: "OperationalInsights", label: "Operational insights" }),
    ];

    const readiness = assessReadiness([provider({ tasks: ["AssistantChat"] })], tasks);

    expect(readiness.uncovered.map((entry) => entry.task)).toEqual([
      "ScopeClassification",
      "OperationalInsights",
    ]);
  });

  it("is not ready at all when no provider exists", () => {
    const readiness = assessReadiness([], [task()]);

    expect(readiness.hasProviders).toBe(false);
    expect(readiness.assistantReady).toBe(false);
  });
});
