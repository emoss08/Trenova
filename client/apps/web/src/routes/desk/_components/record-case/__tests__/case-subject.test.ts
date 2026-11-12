import { describe, expect, it } from "vitest";
import { caseOfThread, threadSubjectOf } from "../case-subject";

describe("caseOfThread", () => {
  it("finds the case a conversation was asked on, for every kind a conversation can be bound to", () => {
    for (const kind of ["shipment", "invoice", "invoice_dispute", "worker"]) {
      const subjectType = threadSubjectOf(kind);

      expect(caseOfThread({ subjectType, subjectId: "rec_1" })).toEqual({ kind, id: "rec_1" });
    }
  });

  it("is null for a conversation about no record, or bound to something that has no case", () => {
    expect(caseOfThread({ subjectType: "", subjectId: null })).toBeNull();
    expect(caseOfThread({ subjectType: "Shipment", subjectId: null })).toBeNull();
    expect(caseOfThread({ subjectType: "AssistantThread", subjectId: "athr_1" })).toBeNull();
  });
});
