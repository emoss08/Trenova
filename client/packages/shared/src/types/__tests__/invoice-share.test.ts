import { describe, expect, it } from "vitest";
import {
  invoiceShareListSchema,
  shareInvoiceFormSchema,
  shareInvoiceResultSchema,
} from "@trenova/shared/types/invoice-share";

const user = {
  id: "usr_dana",
  name: "Dana Whitfield",
  username: "dwhitfield",
  emailAddress: "dana@example.com",
  status: "Active",
  profilePicUrl: "",
  thumbnailUrl: "",
};

const share = {
  id: "invsh_1",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  invoiceId: "inv_1",
  sharedWithId: "usr_dana",
  sharedById: "usr_me",
  note: "",
  tab: "charges",
  shareCount: 2,
  firstSharedAt: 1_788_000_000,
  lastSharedAt: 1_789_000_000,
  createdAt: 1_788_000_000,
  updatedAt: 1_789_000_000,
  sharedWith: user,
  sharedBy: { ...user, id: "usr_me", name: "Marcus Bell" },
};

describe("invoice share schemas", () => {
  it("reads the empty strings Go sends for an unset note and pictures as null", () => {
    const parsed = invoiceShareListSchema.parse({ shares: [share] });

    expect(parsed.shares).toHaveLength(1);
    expect(parsed.shares[0].note).toBeNull();
    expect(parsed.shares[0].sharedWith?.profilePicUrl).toBeNull();
    expect(parsed.shares[0].tab).toBe("charges");
    expect(parsed.shares[0].shareCount).toBe(2);
  });

  it("treats a null share list as empty", () => {
    expect(invoiceShareListSchema.parse({ shares: null }).shares).toEqual([]);
  });

  it("keeps the email delivery outcome of a share", () => {
    const parsed = shareInvoiceResultSchema.parse({
      shares: [share],
      recipientCount: 3,
      emailsQueued: 2,
      emailStatus: "Partial",
    });

    expect(parsed.emailStatus).toBe("Partial");
    expect(parsed.emailsQueued).toBe(2);
  });

  it("rejects a tab or email status the server does not define", () => {
    expect(() =>
      invoiceShareListSchema.parse({ shares: [{ ...share, tab: "nope" }] }),
    ).toThrow();
    expect(() =>
      shareInvoiceResultSchema.parse({
        shares: [],
        recipientCount: 0,
        emailsQueued: 0,
        emailStatus: "Sent",
      }),
    ).toThrow();
  });

  it("requires between one and twenty-five teammates and caps the note", () => {
    expect(shareInvoiceFormSchema.safeParse({ userIds: [], note: "" }).success).toBe(false);
    expect(
      shareInvoiceFormSchema.safeParse({
        userIds: Array.from({ length: 26 }, (_, i) => `usr_${i}`),
        note: "",
      }).success,
    ).toBe(false);
    expect(
      shareInvoiceFormSchema.safeParse({ userIds: ["usr_dana"], note: "x".repeat(1001) }).success,
    ).toBe(false);
    expect(
      shareInvoiceFormSchema.safeParse({ userIds: ["usr_dana"], note: "x".repeat(1000) }).success,
    ).toBe(true);
  });
});
