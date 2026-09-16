import { describe, expect, it } from "vitest";
import { billingQueueItemSchema } from "@trenova/shared/types/billing-queue";
import { shipmentSchema } from "@trenova/shared/types/shipment";

/**
 * The GraphQL billing-queue actions (`BillingQueueActionFields`) and the
 * shipment page's queue-item rows select the payer as a reference:
 * `billToCustomer { id name code }`. The parse boundary must accept that
 * shape, or approving an item throws "Failed to parse BillingQueueItem" the
 * moment a split shipment has a payer.
 */
const actionFieldsItem = {
  id: "bqi_1",
  organizationId: "org_1",
  businessUnitId: "bu_1",
  shipmentId: "shp_1",
  billToCustomerId: "cus_amd",
  allocatedTotalAmount: "120.00",
  billToCustomer: { id: "cus_amd", name: "AMD", code: "AMD" },
  assignedBillerId: null,
  number: "BQ-1",
  status: "Approved",
  billType: "Invoice",
  exceptionReasonCode: null,
  reviewNotes: "",
  exceptionNotes: "",
  reviewStartedAt: null,
  reviewCompletedAt: null,
  canceledById: null,
  canceledAt: null,
  cancelReason: "",
  isAdjustmentOrigin: false,
  sourceInvoiceId: null,
  sourceInvoiceAdjustmentId: null,
  sourceCreditMemoInvoiceId: null,
  correctionGroupId: null,
  rebillStrategy: null,
  requiresReplacementReview: false,
  rerateVariancePercent: "0",
  adjustmentContext: {},
  version: 1,
  createdAt: 1_700_000_000,
  updatedAt: 1_700_000_000,
};

describe("billingQueueItemSchema.billToCustomer", () => {
  it("accepts the customer reference the GraphQL action fragments select", () => {
    const parsed = billingQueueItemSchema.parse(actionFieldsItem);

    expect(parsed.billToCustomer).toEqual({ id: "cus_amd", name: "AMD", code: "AMD" });
  });

  it("still accepts the full customer the REST payload carries", () => {
    const parsed = billingQueueItemSchema.parse({
      ...actionFieldsItem,
      billToCustomer: {
        id: "cus_amd",
        organizationId: "org_1",
        businessUnitId: "bu_1",
        status: "Active",
        code: "AMD",
        name: "AMD",
        addressLine1: "1 Main St",
        addressLine2: null,
        city: "Austin",
        stateId: "st_tx",
        postalCode: "78701",
      },
    });

    expect(parsed.billToCustomer?.name).toBe("AMD");
  });

  it("accepts an absent or null payer", () => {
    expect(
      billingQueueItemSchema.parse({ ...actionFieldsItem, billToCustomer: null }).billToCustomer,
    ).toBeNull();
    const { billToCustomer: _omitted, ...withoutPayer } = actionFieldsItem;
    expect(billingQueueItemSchema.parse(withoutPayer).billToCustomer).toBeUndefined();
  });
});

describe("shipmentSchema.billToCustomer", () => {
  it("accepts a customer reference on the shipment", () => {
    const result = shipmentSchema.shape.billToCustomer.safeParse({
      id: "cus_amd",
      name: "AMD",
      code: "AMD",
    });

    expect(result.success).toBe(true);
  });
});

/**
 * `domain/billingqueue/payershare.go`: the detail read carries the payer's own
 * bill. Decimals are shopspring strings, `method` is "" for a charge billed
 * whole, `percent` may be null, and `additionalChargeId` is "" for freight.
 */
const acmeShare = {
  payerId: "cus_acme",
  isSplit: true,
  payers: [
    { id: "cus_acme", name: "Acme Manufacturing", code: "ACME" },
    { id: "cus_peak", name: "Peak Distributing", code: "PEAK" },
  ],
  lines: [
    {
      kind: "Freight",
      additionalChargeId: "",
      description: "Freight",
      chargeTotal: "2850",
      amount: "1350",
      percent: "47.368421",
      method: "Amount",
      partial: true,
      payers: [
        {
          payerId: "cus_peak",
          payerName: "Peak Distributing",
          payerCode: "PEAK",
          amount: "1500",
          percent: "52.631579",
        },
        {
          payerId: "cus_acme",
          payerName: "Acme Manufacturing",
          payerCode: "ACME",
          amount: "1350",
          percent: "47.368421",
        },
      ],
    },
    {
      kind: "Accessorial",
      additionalChargeId: "ac_det",
      description: "Detention Fee",
      chargeTotal: "1154.38",
      amount: "1154.38",
      percent: null,
      method: "",
      partial: false,
      payers: [
        {
          payerId: "cus_acme",
          payerName: "Acme Manufacturing",
          payerCode: "ACME",
          amount: "1154.38",
          percent: null,
        },
      ],
    },
  ],
  otherPayerLines: [],
  freightAmount: "1350",
  accessorialAmount: "1154.38",
  totalAmount: "2504.38",
  shipmentTotal: "4004.38",
  resolutionError: "",
};

describe("billingQueueItemSchema.payerShare", () => {
  it("keeps the payer's bill as the server resolved it", () => {
    const parsed = billingQueueItemSchema.parse({ ...actionFieldsItem, payerShare: acmeShare });
    const share = parsed.payerShare;

    expect(share?.totalAmount).toBe(2504.38);
    expect(share?.shipmentTotal).toBe(4004.38);
    expect(share?.lines).toHaveLength(2);
    expect(share?.lines[0]).toMatchObject({
      kind: "Freight",
      amount: 1350,
      chargeTotal: 2850,
      method: "Amount",
      partial: true,
    });
    expect(share?.lines[0].additionalChargeId).toBeNull();
    expect(share?.lines[0].payers).toHaveLength(2);
    expect(share?.lines[1].method).toBeNull();
    expect(share?.lines[1].percent).toBeNull();
    expect(share?.otherPayerLines).toEqual([]);
    expect(share?.resolutionError).toBeNull();
  });

  it("carries a split that could not be resolved", () => {
    const parsed = billingQueueItemSchema.parse({
      ...actionFieldsItem,
      payerShare: {
        ...acmeShare,
        lines: [],
        otherPayerLines: [],
        freightAmount: "0",
        accessorialAmount: "0",
        totalAmount: "0",
        payers: [],
        resolutionError: "Amount allocations for this charge must add up to 3000.00",
      },
    });

    expect(parsed.payerShare?.resolutionError).toBe(
      "Amount allocations for this charge must add up to 3000.00",
    );
    expect(parsed.payerShare?.lines).toEqual([]);
  });

  it("is absent on an item read without its shipment", () => {
    expect(billingQueueItemSchema.parse(actionFieldsItem).payerShare ?? null).toBeNull();
  });
});

describe("reassignChargeResultSchema", () => {
  it("parses the queue after a reassignment", async () => {
    const { reassignChargeResultSchema } = await import("@trenova/shared/types/billing-queue");
    const parsed = reassignChargeResultSchema.parse({
      item: {
        ...actionFieldsItem,
        status: "Canceled",
        cancelReason: "Charges reassigned to other payers",
      },
      items: [
        {
          ...actionFieldsItem,
          id: "bqi_2",
          billToCustomerId: "cus_peak",
          status: "ReadyForReview",
        },
      ],
      createdItemIds: ["bqi_2"],
      canceledItemIds: ["bqi_1"],
    });

    expect(parsed.item.status).toBe("Canceled");
    expect(parsed.items.map((row) => row.id)).toEqual(["bqi_2"]);
    expect(parsed.createdItemIds).toEqual(["bqi_2"]);
    expect(parsed.canceledItemIds).toEqual(["bqi_1"]);
  });
});
