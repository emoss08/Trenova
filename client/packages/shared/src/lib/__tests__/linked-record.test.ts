import { isReferenceField, linkedRecordSchema } from "@trenova/shared/lib/linked-record";
import { billingQueueItemSchema } from "@trenova/shared/types/billing-queue";
import { documentSchema } from "@trenova/shared/types/document";
import { invoiceEmailAttemptSchema, invoiceSchema } from "@trenova/shared/types/invoice";
import { shipmentSchema } from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";
import { z } from "zod";

// What the server sends for a linked record of a kind the reader cannot read: the view's fields,
// and null for every other field it would have written (services/tms/internal/api/responseguard).
function withheld(view: Record<string, unknown>, others: readonly string[]) {
  return { ...view, ...Object.fromEntries(others.map((field) => [field, null])) };
}

describe("linkedRecordSchema", () => {
  const person = z.object({
    id: z.string(),
    name: z.string(),
    emailAddress: z.email(),
    timezone: z.string().min(1),
  });

  it("accepts null for a field outside the kind's reference view", () => {
    const parsed = linkedRecordSchema("user", person).parse({
      id: "usr_1",
      name: "Billie",
      emailAddress: null,
      timezone: null,
    });

    expect(parsed).toEqual({ id: "usr_1", name: "Billie", emailAddress: null, timezone: null });
  });

  it("parses a whole record for a reader of its kind as its own schema does", () => {
    const record = { id: "usr_1", name: "Billie", emailAddress: "b@example.com", timezone: "UTC" };

    expect(linkedRecordSchema("user", person).parse(record)).toEqual(person.parse(record));
  });

  it("keeps a reference field as strict as the record's own schema", () => {
    const result = linkedRecordSchema("user", person).safeParse({
      id: "usr_1",
      name: null,
      emailAddress: null,
      timezone: null,
    });

    expect(result.success).toBe(false);
  });

  it("shows only bookkeeping fields of a kind with no view of its own", () => {
    expect(isReferenceField("distance_profile", "id")).toBe(true);
    expect(isReferenceField("distance_profile", "name")).toBe(false);
    expect(isReferenceField("user", "name")).toBe(true);
    expect(isReferenceField("user", "emailAddress")).toBe(false);
  });

  it("shows a document's name, type and size, never where or how it is stored", () => {
    expect(isReferenceField("document", "originalName")).toBe(true);
    expect(isReferenceField("document", "fileType")).toBe(true);
    expect(isReferenceField("document", "fileSize")).toBe(true);
    expect(isReferenceField("document", "createdAt")).toBe(true);
    expect(isReferenceField("document", "fileName")).toBe(false);
    expect(isReferenceField("document", "storagePath")).toBe(false);
    expect(isReferenceField("document", "checksumSha256")).toBe(false);
  });

  it("shows an email's delivery state, never its recipients, subject or error", () => {
    expect(isReferenceField("email_log", "status")).toBe(true);
    expect(isReferenceField("email_log", "sentAt")).toBe(true);
    expect(isReferenceField("email_log", "failedAt")).toBe(true);
    expect(isReferenceField("email_log", "providerMessageId")).toBe(true);
    expect(isReferenceField("email_log", "toRecipients")).toBe(false);
    expect(isReferenceField("email_log", "subject")).toBe(false);
    expect(isReferenceField("email_log", "lastError")).toBe(false);
    expect(isReferenceField("email_profile", "name")).toBe(true);
    expect(isReferenceField("email_profile", "senderEmail")).toBe(false);
  });
});

describe("REST records carried inside another", () => {
  it("parses a billing queue item whose biller and shipment the reader cannot read", () => {
    const shape = billingQueueItemSchema.shape;

    const biller = shape.assignedBiller.parse(
      withheld({ id: "usr_1", name: "Billie", username: "billie" }, [
        "status",
        "emailAddress",
        "timezone",
        "timeFormat",
        "locale",
        "isLocked",
        "mustChangePassword",
        "currentOrganizationId",
      ]),
    );
    expect(biller?.name).toBe("Billie");
    expect(biller?.emailAddress).toBeNull();

    const shipment = shape.shipment.parse(
      withheld(
        { id: "shp_1", proNumber: "PRO-9", bol: "BOL-9", status: "Completed", customerId: "cus_1" },
        [
          "serviceTypeId",
          "shipmentTypeId",
          "moves",
          "customer",
          "commodities",
          "additionalCharges",
          "chargeAllocations",
          "billingSplitSummary",
          "totalChargeAmount",
          "weight",
        ],
      ),
    );
    expect(shipment?.proNumber).toBe("PRO-9");
    expect(shipment?.moves).toBeNull();
    expect(shipment?.customer).toBeNull();
  });

  it("parses an invoice whose customer, shipment and PDF the reader cannot read", () => {
    const shape = invoiceSchema.shape;

    const customer = shape.customer.parse(
      withheld(
        {
          id: "cus_1",
          code: "ACME",
          name: "Acme Foods",
          status: "Active",
          addressLine1: "1 Market Street",
          city: "Dayton",
          stateId: "us_oh",
          postalCode: "45402",
        },
        [
          "brokerVettingEnabled",
          "statusUpdatePreference",
          "billingProfile",
          "emailProfile",
          "dotNumber",
        ],
      ),
    );
    expect(customer?.name).toBe("Acme Foods");
    expect(customer?.billingProfile).toBeNull();

    const shipper = shape.shipperCustomer.parse(
      withheld(
        {
          id: "cus_2",
          code: "SHIP",
          name: "Shipper Co",
          status: "Active",
          addressLine1: "2 Dock Road",
          city: "Toledo",
          stateId: "us_oh",
          postalCode: "43604",
        },
        ["billingProfile"],
      ),
    );
    expect(shipper?.name).toBe("Shipper Co");

    const shipment = shape.shipment.parse(
      withheld({ id: "shp_1", proNumber: "PRO-1", customerId: "cus_1" }, [
        "moves",
        "serviceTypeId",
        "customer",
      ]),
    );
    expect(shipment?.moves).toBeNull();

    const pdf = shape.pdfDocument.parse(
      withheld(
        {
          id: "doc_1",
          organizationId: "org_1",
          businessUnitId: "bu_1",
          version: 1,
          createdAt: 1_700_000_000,
          updatedAt: 1_700_000_000,
          originalName: "INV-1.pdf",
          fileType: "application/pdf",
          fileSize: 48_213,
        },
        ["fileName", "storagePath", "status", "resourceType", "resourceId", "hasExtractedText"],
      ),
    );
    expect(pdf?.originalName).toBe("INV-1.pdf");
    expect(pdf?.fileSize).toBe(48_213);
    expect(pdf?.storagePath).toBeNull();

    const queueItem = shape.billingQueueItem.parse(
      withheld(
        {
          id: "bqi_1",
          organizationId: "org_1",
          businessUnitId: "bu_1",
          version: 1,
          createdAt: 1_700_000_000,
          updatedAt: 1_700_000_000,
          number: "BQ-1",
          status: "ReadyForReview",
        },
        ["shipmentId", "billType", "billToCustomerId", "reviewNotes", "exceptionNotes"],
      ),
    );
    expect(queueItem?.number).toBe("BQ-1");
    expect(queueItem?.status).toBe("ReadyForReview");
    expect(queueItem?.reviewNotes).toBeNull();
  });

  it("parses an email attempt whose email log the reader cannot read", () => {
    const email = invoiceEmailAttemptSchema.shape.email.parse(
      withheld(
        {
          id: "em_1",
          status: "Delivered",
          sentAt: 1_700_000_000,
          providerMessageId: "msg_1",
          createdAt: 1_700_000_000,
          updatedAt: 1_700_000_000,
        },
        ["toRecipients", "subject", "lastError", "fromEmail", "profile"],
      ),
    );
    expect(email?.status).toBe("Delivered");
    expect(email?.toRecipients).toBeNull();
    expect(email?.lastError).toBeNull();
  });

  it("keeps every key of the record's own schema", () => {
    const linked = linkedRecordSchema("document", documentSchema);

    expect(Object.keys(linked.shape).sort()).toEqual(Object.keys(documentSchema.shape).sort());
  });

  it("keeps the shipment's own reference fields required where its schema requires them", () => {
    const result = linkedRecordSchema("shipment", shipmentSchema).safeParse({ proNumber: 7 });

    expect(result.success).toBe(false);
  });
});
