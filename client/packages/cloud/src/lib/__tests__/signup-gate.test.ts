import { describe, expect, it } from "vitest";
import { signupRedirect } from "../signup-gate";

describe("signupRedirect", () => {
  it("lets the signup pages render only on cloud with signup on", () => {
    expect(signupRedirect({ platformMode: "cloud", signupEnabled: true })).toBeNull();
  });

  it("sends everybody else to sign-in", () => {
    expect(signupRedirect({ platformMode: "cloud", signupEnabled: false })).toBe("/login");
    expect(signupRedirect({ platformMode: "self_hosted", signupEnabled: true })).toBe("/login");
    expect(signupRedirect(undefined)).toBe("/login");
  });
});
