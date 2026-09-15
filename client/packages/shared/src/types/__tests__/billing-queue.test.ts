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
