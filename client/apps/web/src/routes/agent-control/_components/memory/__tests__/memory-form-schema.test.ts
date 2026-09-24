import { describe, expect, it } from "vitest";
import {
  MEMORY_CONTENT_LIMIT,
  memoryFormDefaults,
  memoryFormSchema,
  toMemoryInput,
} from "../memory-form-schema";

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
    expect(MEMORY_CONTENT_LIMIT).toBe(4000);
    expect(memoryFormSchema.safeParse({ ...memoryFormDefaults, content: "" }).success).toBe(false);
    expect(
      memoryFormSchema.safeParse({
        ...memoryFormDefaults,
        content: "x".repeat(MEMORY_CONTENT_LIMIT),
      }).success,
    ).toBe(true);

    const over = memoryFormSchema.safeParse({
      ...memoryFormDefaults,
      content: "x".repeat(MEMORY_CONTENT_LIMIT + 1),
    });
    expect(over.success).toBe(false);
    expect(over.error?.issues[0]?.message).toBe("Keep it to 4000 characters");
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
