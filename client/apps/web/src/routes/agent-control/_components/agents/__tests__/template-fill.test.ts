import type { AgentTemplate } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { agentFormDefaults } from "../agent-form-schema";
import { applyTemplateStarter } from "../template-fill";

const starter: AgentTemplate = {
  template: "DispatchAssignment",
  label: "Dispatch coverage",
  description: "Covers moves with no driver.",
  starterInstructions: "Review moves with no driver and propose assignments.",
  starterTools: ["get_shipment", "search_worker", "assign_move"],
  starterTrigger: "Scheduled",
  starterEvents: [],
  starterCron: "*/30 * * * *",
  starterCeiling: "ActWithApproval",
  starterDataAccess: "Internal",
  starterOutput: "Report",
  starterDailyRunLimit: 0,
  systemKey: "",
  contextProviders: ["Organization", "Clock", "Tools"],
};

/**
 * A template is a starting point, never a restriction: it fills what the
 * person has not written yet and leaves everything they already typed alone.
 */
describe("applyTemplateStarter", () => {
  it("fills every pristine field from the starter", () => {
    const patch = applyTemplateStarter(agentFormDefaults, starter);

    expect(patch).toEqual({
      template: "DispatchAssignment",
      instructions: starter.starterInstructions,
      toolNames: starter.starterTools,
      triggerMode: "Scheduled",
      cronExpression: "*/30 * * * *",
      autonomyCeiling: "ActWithApproval",
      dataAccessCeiling: "Internal",
      outputMode: "Report",
      contextProviders: ["Organization", "Clock", "Tools"],
    });
  });

  it("never overwrites instructions or tools the person already chose", () => {
    const patch = applyTemplateStarter(
      { ...agentFormDefaults, instructions: "Our own rules.", toolNames: ["get_worker"] },
      starter,
    );

    expect(patch.instructions).toBeUndefined();
    expect(patch.toolNames).toBeUndefined();
    expect(patch.template).toBe("DispatchAssignment");
  });

  it("does not switch a trigger the person already set away from chat", () => {
    const patch = applyTemplateStarter({ ...agentFormDefaults, triggerMode: "Event" }, starter);

    expect(patch.triggerMode).toBeUndefined();
    expect(patch.cronExpression).toBeUndefined();
  });

  it("sets the data access a template needs, whatever the form held", () => {
    const patch = applyTemplateStarter(
      { ...agentFormDefaults, dataAccessCeiling: "Internal" },
      { ...starter, template: "CashApplication", starterDataAccess: "Restricted" },
    );

    expect(patch.dataAccessCeiling).toBe("Restricted");
  });

  it("caps the runs a day an agent with no cap of its own may start", () => {
    const analyst = { ...starter, template: "InsightAnalyst" as const, starterDailyRunLimit: 20 };

    expect(applyTemplateStarter(agentFormDefaults, analyst).dailyRunLimit).toBe(20);
    expect(
      applyTemplateStarter({ ...agentFormDefaults, dailyRunLimit: 5 }, analyst).dailyRunLimit,
    ).toBeUndefined();
    expect(applyTemplateStarter(agentFormDefaults, starter).dailyRunLimit).toBeUndefined();
  });

  it("only sets the template when cleared", () => {
    expect(applyTemplateStarter(agentFormDefaults, null)).toEqual({ template: null });
  });
});
