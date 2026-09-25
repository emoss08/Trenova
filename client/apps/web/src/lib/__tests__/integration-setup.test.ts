import { describe, expect, it } from "vitest";
import { integrationSetupStepStates } from "../integration-setup";

const steps = [
  { id: "connect", label: "Connect" },
  { id: "review", label: "Review company" },
  { id: "map", label: "Map accounts" },
] as const;

describe("integrationSetupStepStates", () => {
  it("marks the steps before the active one done and the ones after it pending", () => {
    expect(integrationSetupStepStates(steps, "review").map((step) => step.state)).toEqual([
      "done",
      "active",
      "pending",
    ]);
  });

  it("starts at the first step", () => {
    expect(integrationSetupStepStates(steps, "connect").map((step) => step.state)).toEqual([
      "active",
      "pending",
      "pending",
    ]);
  });

  it("marks every earlier step done on the last one", () => {
    expect(integrationSetupStepStates(steps, "map").map((step) => step.state)).toEqual([
      "done",
      "done",
      "active",
    ]);
  });

  it("falls back to the first step for an id the wizard does not have", () => {
    expect(integrationSetupStepStates(steps, "gone").map((step) => step.state)).toEqual([
      "active",
      "pending",
      "pending",
    ]);
  });

  it("keeps each step's id, label and detail", () => {
    expect(
      integrationSetupStepStates(
        [{ id: "connect", label: "Connect", detail: "Sign in" }],
        "connect",
      ),
    ).toEqual([{ id: "connect", label: "Connect", detail: "Sign in", state: "active" }]);
  });
});
