import { goEnumValues } from "@/test/go-source";
import { describe, expect, it } from "vitest";
import {
  MAILBOX_STATUSES,
  MIN_CONFIDENCE_FLOOR,
  PROVIDERS,
  REVIEW_POLICIES,
  mailboxFormSchema,
  mailboxInput,
  newMailboxDefaults,
} from "../mailbox-schema";

const valid = {
  ...newMailboxDefaults(),
  name: "Tenders",
  address: "tenders@acme-logistics.com",
};

describe("mailboxFormSchema", () => {
  it("starts a new mailbox reviewing everything, as the server defaults it", () => {
    // A mailbox somebody has not finished configuring should not act alone.
    expect(newMailboxDefaults().reviewPolicy).toBe("AlwaysReview");
    expect(newMailboxDefaults().status).toBe("Active");
  });

  it("accepts a complete mailbox", () => {
    expect(mailboxFormSchema.safeParse(valid).success).toBe(true);
  });

  it("requires a real address", () => {
    const result = mailboxFormSchema.safeParse({ ...valid, address: "not an address" });
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path).toEqual(["address"]);
  });

  it("holds a confidence bar to the server's floor only when the policy reads it", () => {
    const below = { ...valid, reviewPolicy: "ReviewBelowConfidence", minConfidence: 0.3 } as const;
    const result = mailboxFormSchema.safeParse(below);
    expect(result.success).toBe(false);
    expect(result.error?.issues[0]?.path).toEqual(["minConfidence"]);

    expect(
      mailboxFormSchema.safeParse({ ...below, minConfidence: MIN_CONFIDENCE_FLOOR }).success,
    ).toBe(true);
    expect(mailboxFormSchema.safeParse({ ...valid, minConfidence: 0.3 }).success).toBe(true);
  });
});

describe("mailboxInput", () => {
  it("sends an empty purpose as absent and trims what was typed", () => {
    const input = mailboxInput({ ...valid, name: "  Tenders ", purpose: "   " });
    expect(input.name).toBe("Tenders");
    expect(input.purpose).toBeNull();
  });
});

/*
The selects are drawn from these lists. A provider or policy added in Go and
missing here is one nobody can choose — and a mailbox already set to it fails
the form's parse the moment somebody opens it to edit.
*/
describe("mailbox option lists", () => {
  const file = "services/tms/internal/core/domain/inboundmessage/enums.go";

  it("cover every provider, review policy and status the server has", () => {
    expect([...PROVIDERS].sort()).toEqual(goEnumValues({ file, typeName: "Provider" }).sort());
    expect([...REVIEW_POLICIES].sort()).toEqual(
      goEnumValues({
        file,
        typeName: "ReviewPolicy",
        listFn: "AllReviewPolicies",
        constPrefix: "Review",
      }).sort(),
    );
    expect([...MAILBOX_STATUSES].sort()).toEqual(
      goEnumValues({
        file,
        typeName: "MailboxStatus",
        listFn: "AllMailboxStatuses",
        constPrefix: "Mailbox",
      }).sort(),
    );
  });
});
