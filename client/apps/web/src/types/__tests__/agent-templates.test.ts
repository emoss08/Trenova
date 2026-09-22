import { goEnumValues } from "@/test/go-source";
import { agentTemplateKindSchema } from "@/types/assistant";
import { describe, expect, it } from "vitest";

/**
 * Every template the server offers has to be nameable here.
 *
 * The agent list, the agent form and every proposal card parse their payload
 * through this enum, so a template the client has not heard of does not
 * degrade to an unknown row — it throws the whole response away and AI
 * Control goes blank, exactly as it did when DailyBriefing landed on the
 * server ahead of aiTaskSchema.
 *
 * This reads AllTemplates() out of the Go source, so a template added on the
 * server without being added here fails this test rather than the page.
 */
function serverTemplates(): string[] {
  return goEnumValues({
    file: "services/tms/internal/core/domain/agentdefinition/enums.go",
    typeName: "Template",
  });
}

describe("agentTemplateKindSchema", () => {
  const templates = serverTemplates();

  it("finds the templates the server offers", () => {
    expect(templates.length).toBeGreaterThan(10);
  });

  it.each(templates)("accepts the %s template the server offers", (template) => {
    expect(agentTemplateKindSchema.safeParse(template).success).toBe(true);
  });

  it("names no template the server does not offer", () => {
    expect([...agentTemplateKindSchema.options].sort()).toEqual([...templates].sort());
  });

  it("still refuses a template the server does not offer", () => {
    expect(agentTemplateKindSchema.safeParse("TelepathyDesk").success).toBe(false);
  });
});
