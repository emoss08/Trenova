import { aiProviderSchema } from "@/types/ai-provider";
import { describe, expect, it } from "vitest";

/**
 * A provider's `Tasks` is a `[]Task` with `nullzero` on the Go side, so a
 * provider that has not been assigned any task comes back as `"tasks": null`.
 * The readiness banner and the provider list both index into `tasks`, so the
 * schema has to turn that null into an empty list rather than refuse the row.
 */
const unassignedProvider = {
  id: "aiprv_01JPROVIDER000000000000",
  businessUnitId: "bu_01JBUSINESSUNIT000000000",
  organizationId: "org_01JORGANIZATION00000000",
  name: "Local Ollama",
  description: "",
  kind: "Ollama",
  baseUrl: "http://127.0.0.1:11434",
  model: "llama3.1",
  allowPrivateNetwork: true,
  structuredOutputMode: "JSONSchema",
  maxTokens: 8192,
  tasks: null,
  priority: 100,
  trusted: false,
  enabled: false,
  version: 1,
  createdAt: 1_758_000_000,
  updatedAt: 1_758_000_100,
};

describe("aiProviderSchema", () => {
  it("reads a provider whose tasks the server wrote as null", () => {
    expect(aiProviderSchema.parse(unassignedProvider).tasks).toEqual([]);
  });

  it("keeps the tasks of a provider that has some", () => {
    const parsed = aiProviderSchema.parse({
      ...unassignedProvider,
      tasks: ["AssistantChat", "General"],
    });

    expect(parsed.tasks).toEqual(["AssistantChat", "General"]);
  });
});
