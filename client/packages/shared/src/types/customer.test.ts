import { describe, expect, it } from "vitest";
import { customerSchema } from "./customer";

function customer(overrides: Record<string, unknown> = {}) {
  return {
    status: "Active",
    code: "ACME",
    name: "Acme Logistics",
    addressLine1: "1 Main St",
    city: "Dallas",
    stateId: "us_tx",
    postalCode: "75201",
    ...overrides,
  };
}

function messages(input: Record<string, unknown>): string[] {
  const result = customerSchema.safeParse(input);
  return result.success ? [] : result.error.issues.map((issue) => issue.message);
}

describe("customerSchema broker vetting fields", () => {
  it("defaults broker vetting off and leaves DOT and MC numbers empty", () => {
    const parsed = customerSchema.parse(customer());
    expect(parsed.brokerVettingEnabled).toBe(false);
    expect(parsed.dotNumber ?? null).toBeNull();
    expect(parsed.mcNumber ?? null).toBeNull();
  });

  it("treats blank DOT and MC numbers as absent", () => {
    const parsed = customerSchema.parse(customer({ dotNumber: "", mcNumber: "" }));
    expect(parsed.dotNumber).toBeNull();
    expect(parsed.mcNumber).toBeNull();
  });

  it("accepts digit-only DOT and MC numbers", () => {
    const parsed = customerSchema.parse(
      customer({ dotNumber: "1234567", mcNumber: "654321", brokerVettingEnabled: true }),
    );
    expect(parsed.dotNumber).toBe("1234567");
    expect(parsed.mcNumber).toBe("654321");
    expect(parsed.brokerVettingEnabled).toBe(true);
  });

  it("rejects DOT and MC numbers with anything but digits", () => {
    expect(messages(customer({ dotNumber: "USDOT 123" }))).toContain(
      "DOT number must contain only digits (12 max)",
    );
    expect(messages(customer({ mcNumber: "MC-654321" }))).toContain(
      "MC number must contain only digits (12 max)",
    );
  });

  it("rejects numbers longer than 12 digits", () => {
    expect(messages(customer({ dotNumber: "1234567890123" }))).toContain(
      "DOT number must contain only digits (12 max)",
    );
  });

  it("requires a DOT number when broker vetting is on", () => {
    const result = customerSchema.safeParse(customer({ brokerVettingEnabled: true }));
    expect(result.success).toBe(false);
    const issue = result.error?.issues.find(
      (candidate) =>
        candidate.message === "A DOT number is required to vet this customer as a broker",
    );
    expect(issue?.path).toEqual(["dotNumber"]);
  });
});
