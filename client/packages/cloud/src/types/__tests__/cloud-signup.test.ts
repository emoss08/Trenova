import { describe, expect, it } from "vitest";
import {
  passwordMatchesEmail,
  passwordRequirements,
  passwordStrength,
  SIGNUP_FORM_DEFAULTS,
  signupFormSchema,
  type SignupFormValues,
} from "../cloud-signup";

const valid: SignupFormValues = {
  name: "Jordan Rivera",
  emailAddress: "jordan@riverafreight.com",
  companyName: "Rivera Freight LLC",
  password: "correct horse battery",
  acceptTerms: true,
  website: "",
};

function issuesFor(values: Partial<SignupFormValues>) {
  const result = signupFormSchema.safeParse({ ...valid, ...values });
  if (result.success) {
    return {};
  }
  return Object.fromEntries(
    result.error.issues.map((issue) => [issue.path.join("."), issue.message]),
  );
}

describe("signupFormSchema", () => {
  it("accepts a complete signup", () => {
    expect(signupFormSchema.safeParse(valid).success).toBe(true);
  });

  it("rejects the empty defaults field by field", () => {
    const issues = issuesFor(SIGNUP_FORM_DEFAULTS);
    expect(Object.keys(issues).sort()).toEqual(
      ["acceptTerms", "companyName", "emailAddress", "name", "password"].sort(),
    );
  });

  it("requires 12 characters, counting exactly", () => {
    expect(issuesFor({ password: "a".repeat(11) }).password).toBe(
      "Password must be at least 12 characters",
    );
    expect(issuesFor({ password: "abcdefghijk1" }).password).toBeUndefined();
  });

  it("refuses the email address as the password, in any case", () => {
    expect(
      issuesFor({
        emailAddress: "jordan.rivera@example.com",
        password: "Jordan.Rivera@Example.com",
      }).password,
    ).toBe("Password must not be your email address");
  });

  it("refuses the email's local part as the password", () => {
    expect(
      issuesFor({ emailAddress: "jordanrivera1@example.com", password: "JordanRivera1" }).password,
    ).toBe("Password must not be your email address");
  });

  it("does not let whitespace-only names or companies through", () => {
    const issues = issuesFor({ name: "   ", companyName: "\t" });
    expect(issues.name).toBe("Enter your name");
    expect(issues.companyName).toBe("Enter your company name");
  });

  it("validates the email after trimming it", () => {
    expect(
      issuesFor({ emailAddress: "  jordan@riverafreight.com  " }).emailAddress,
    ).toBeUndefined();
    expect(issuesFor({ emailAddress: "not-an-email" }).emailAddress).toBe(
      "Enter a valid email address",
    );
  });

  it("requires the terms to be accepted", () => {
    expect(issuesFor({ acceptTerms: false }).acceptTerms).toBe(
      "Accept the terms of service and privacy policy to continue",
    );
  });

  it("does not block on a filled honeypot — the server drops those quietly", () => {
    expect(signupFormSchema.safeParse({ ...valid, website: "http://spam.example" }).success).toBe(
      true,
    );
  });
});

describe("password hints", () => {
  it("reports each requirement as it is met", () => {
    expect(passwordRequirements("", "jordan@example.com").map((r) => r.met)).toEqual([
      false,
      false,
    ]);
    expect(passwordRequirements("jordan", "jordan@example.com").map((r) => r.met)).toEqual([
      false,
      false,
    ]);
    expect(
      passwordRequirements("a long passphrase", "jordan@example.com").map((r) => r.met),
    ).toEqual([true, true]);
  });

  it("does not match an empty email against an empty password", () => {
    expect(passwordMatchesEmail("", "")).toBe(false);
    expect(passwordMatchesEmail("something", "")).toBe(false);
  });

  it("scores longer and more varied passwords higher, and the email as zero", () => {
    const email = "jordan@example.com";
    expect(passwordStrength("", email)).toBe(0);
    expect(passwordStrength("jordan@example.com", email)).toBe(0);
    expect(passwordStrength("abcdefghijkl", email)).toBe(1);
    expect(passwordStrength("Abcdefgh1jklmnop", email)).toBeGreaterThan(
      passwordStrength("abcdefghijkl", email),
    );
    expect(passwordStrength("Tr3nova!Freight#Lanes", email)).toBe(4);
    expect(passwordStrength("aaaaaaaaaaaaaaaaaaaa", email)).toBeLessThan(2);
  });
});
