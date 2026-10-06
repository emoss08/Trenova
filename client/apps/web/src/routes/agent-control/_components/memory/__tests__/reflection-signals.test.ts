import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { reflectionSignalLabel } from "../reflection-signals";

const t = ((text: string) => text) as TranslateFn;

/*
The kinds are agent.ReflectionSignalKind in
services/tms/internal/core/domain/agent/agentreflection.go. A memory's evidence
carries them as plain strings, so a kind the client has not been taught yet
must still show, as it was recorded.
*/
describe("reflectionSignalLabel", () => {
  it("names every kind the server records", () => {
    const kinds = [
      "ToolRecovered",
      "ToolFailed",
      "PersonCorrected",
      "StandingRequest",
      "ProposalModified",
      "ProposalRejected",
      "NegativeFeedback",
      "LongTask",
    ];
    const labels = kinds.map((kind) => reflectionSignalLabel(kind, t));

    expect(labels).toEqual([
      "A tool worked after failing",
      "A tool failed",
      "A person corrected it",
      "A person said how they want it done",
      "A proposal was changed",
      "A proposal was refused",
      "A reply was rated unhelpful",
      "A long task",
    ]);
  });

  it("shows a kind it does not know as it was recorded", () => {
    expect(reflectionSignalLabel("SomethingNew", t)).toBe("SomethingNew");
  });
});
