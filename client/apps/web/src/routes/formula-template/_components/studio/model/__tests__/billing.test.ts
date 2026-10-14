import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { billingLabel } from "../billing";

const t = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d+)\}/g, (_m, index) => String(args[Number(index)]))) as TranslateFn;

const billingV5 = { versionNumber: 5, effectiveFrom: 1_760_000_000 };

describe("billingLabel", () => {
  it("names the approved content that keeps billing while a revision is reviewed", () => {
    expect(
      billingLabel({ status: "InReview", currentVersionNumber: 6, billing: billingV5 }, t),
    ).toBe("Billing v5 while v6 is in review");
  });

  it("names it while the revision is still a draft", () => {
    expect(billingLabel({ status: "Draft", currentVersionNumber: 7, billing: billingV5 }, t)).toBe(
      "Billing v5 while v7 is a draft",
    );
  });

  it("says an approval scheduled for later has not started billing", () => {
    expect(billingLabel({ status: "Active", currentVersionNumber: 6, billing: billingV5 }, t)).toBe(
      "Billing v5 until v6 takes effect",
    );
  });

  it("says nothing bills until a deferred first approval starts", () => {
    expect(
      billingLabel(
        {
          status: "Active",
          currentVersionNumber: 1,
          billing: { versionNumber: 1, effectiveFrom: 4_070_908_800, pending: true },
        },
        t,
      ),
    ).toBe("Nothing bills until v1 takes effect");
  });

  it("says nothing when the content shown is the content billing", () => {
    expect(
      billingLabel({ status: "Active", currentVersionNumber: 5, billing: billingV5 }, t),
    ).toBeNull();
  });

  it("says nothing when nothing bills", () => {
    expect(billingLabel({ status: "Draft", currentVersionNumber: 1, billing: null }, t)).toBeNull();
    expect(
      billingLabel({ status: "Draft", currentVersionNumber: 1, billing: undefined }, t),
    ).toBeNull();
    expect(
      billingLabel({ status: "Inactive", currentVersionNumber: 6, billing: billingV5 }, t),
    ).toBeNull();
  });
});
