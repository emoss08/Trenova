import { describe, expect, it } from "vitest";
import { memoryFormDefaults, memoryFormSchema, toMemoryInput } from "../memory-form-schema";

describe("memoryFormSchema", () => {
  it("accepts an organization-wide instruction with nothing else set", () => {
    const parsed = memoryFormSchema.safeParse({
      ...memoryFormDefaults,
      kind: "Instruction",
      content: "  Every customer gets a POD within a day.  ",
    });

    expect(parsed.success).toBe(true);
    expect(parsed.data?.content).toBe("Every customer gets a POD within a day.");
  });

  // Half a subject names nothing: a type without a record, or a record the
  // server cannot look up. The form points at the missing half.
  it("refuses a subject type without its record, and the reverse", () => {
    const noRecord = memoryFormSchema.safeParse({
      ...memoryFormDefaults,
      content: "x",
      subjectType: "Customer",
    });
    expect(noRecord.success).toBe(false);
    expect(noRecord.error?.issues.map((issue) => issue.path.join("."))).toContain("subjectId");

    const noType = memoryFormSchema.safeParse({
      ...memoryFormDefaults,
      content: "x",
      subjectId: "cus_1",
    });
    expect(noType.success).toBe(false);
    expect(noType.error?.issues.map((issue) => issue.path.join("."))).toContain("subjectType");
  });

  it("bounds the content the way the server does", () => {
    expect(memoryFormSchema.safeParse({ ...memoryFormDefaults, content: "" }).success).toBe(false);
    expect(
      memoryFormSchema.safeParse({ ...memoryFormDefaults, content: "x".repeat(2001) }).success,
    ).toBe(false);
  });
});

describe("toMemoryInput", () => {
  it("sends absent for what was not set rather than empty strings", () => {
    expect(
      toMemoryInput({
        kind: "Fact",
        content: "The yard closes at 18:00.",
        subjectType: null,
        subjectId: null,
        toolName: "",
        expiresAt: null,
        version: 3,
      }),
    ).toEqual({
      kind: "Fact",
      content: "The yard closes at 18:00.",
      subjectType: null,
      subjectId: null,
      toolName: null,
      expiresAt: null,
      version: 3,
    });
  });

  it("carries a subject and an expiry through", () => {
    expect(
      toMemoryInput({
        kind: "Instruction",
        content: "Needs the POD within a day.",
        subjectType: "Customer",
        subjectId: "cus_1",
        toolName: " assign_move ",
        expiresAt: 1_800_000_000,
        version: 0,
      }),
    ).toEqual({
      kind: "Instruction",
      content: "Needs the POD within a day.",
      subjectType: "Customer",
      subjectId: "cus_1",
      toolName: "assign_move",
      expiresAt: 1_800_000_000,
      version: 0,
    });
  });
});
