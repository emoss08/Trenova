import { beforeEach, describe, expect, it, vi } from "vitest";
import type { AccessorialCharge } from "@trenova/shared/types/accessorial-charge";
import type { Customer } from "@trenova/shared/types/customer";
import type { FleetCode } from "@trenova/shared/types/fleet-code";
import type { HoldReason } from "@/types/hold-reason";
import { AccessorialChargeService } from "../accessorial-charge";
import { CustomerService } from "../customer";
import { FleetCodeService } from "../fleet-code";
import { HoldReasonService } from "../hold-reason";

const requestGraphQL = vi.fn();

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: (...args: unknown[]) => requestGraphQL(...args),
}));

function sentVariables() {
  expect(requestGraphQL).toHaveBeenCalledTimes(1);
  return requestGraphQL.mock.calls[0]?.[0] as {
    operationName: string;
    variables: Record<string, unknown>;
  };
}

const customer = {
  status: "Active",
  code: "ACME",
  name: "Acme Freight",
  addressLine1: "1 Main St",
  addressLine2: null,
  city: "Columbus",
  stateId: "us_oh",
  postalCode: "43004",
  isGeocoded: false,
  longitude: -82.9,
  latitude: 39.9,
  placeId: "place_1",
  externalId: null,
  dotNumber: "1234567",
  mcNumber: null,
  brokerVettingEnabled: true,
  checkCallIntervalHours: 4,
  etaRiskThresholdMinutes: null,
  allowConsolidation: true,
  exclusiveConsolidation: false,
  consolidationPriority: 1,
  statusUpdatePreference: "Arrivals",
  statusUpdateRecipients: "ops@acme.test",
  version: 3,
  billingProfile: {
    invoiceDelivery: "PerShipment",
    billingCycle: "Monthly",
    billingCycleAnchorDay: 1,
    billingCycleTimezone: "UTC",
    lastBilledPeriodEnd: null,
    paymentTerm: "Net30",
    hasBillingControlOverrides: false,
    creditLimit: 25000.5,
    creditBalance: 1200,
    creditStatus: "Active",
    enforceCreditLimit: true,
    autoCreditHold: false,
    creditHoldReason: "",
    creditHoldAutomatic: false,
    creditReviewedAt: null,
    autoSendInvoiceOnGeneration: true,
    sendPaymentReminders: true,
    sendStatements: true,
    factoringCompanyId: null,
    emailInvoiceEnabled: true,
    ediInvoiceEnabled: false,
    splitBy: "Customer",
    sectionBy: "Shipment",
    invoiceDetail: "Detailed",
    minConsolidatedAmount: null,
    maxShipmentsPerInvoice: 0,
    invoiceNumberFormat: "Default",
    customerInvoicePrefix: "",
    invoiceCopies: 1,
    revenueAccountId: "gla_rev",
    arAccountId: null,
    applyLateCharges: false,
    lateChargeRate: 1.5,
    gracePeriodDays: 0,
    taxExempt: false,
    taxExemptNumber: "",
    enforceCustomerBillingReq: true,
    validateCustomerRates: true,
    autoTransfer: true,
    autoMarkReadyToBill: true,
    autoApprove: false,
    autoBill: true,
    countLateOnlyOnAppointmentStops: false,
    autoApplyAccessorials: true,
    fuelSurchargeMode: "None",
    fuelSurchargeProgramId: null,
    billingCurrency: "USD",
    requirePONumber: true,
    requireBOLNumber: false,
    requireDeliveryNumber: false,
    defaultBillerId: null,
    billingNotes: "",
    documentTypes: [
      { id: "dt_pod", code: "POD" },
      { id: "dt_bol", code: "BOL" },
    ],
  },
  emailProfile: {
    subject: "Invoice",
    comment: "",
    toRecipients: "ap@acme.test",
    ccRecipients: "",
    bccRecipients: "",
    attachmentName: "",
    readReceipt: false,
    includeShipmentDetail: true,
  },
} as Customer;

describe("CustomerService", () => {
  beforeEach(() => requestGraphQL.mockReset());

  it("updates a customer with its profiles, sending decimals as strings and document types as ids", async () => {
    requestGraphQL.mockResolvedValue({ updateCustomer: { id: "cus_1", version: 4 } });

    const result = await new CustomerService().update("cus_1", customer);

    const { operationName, variables } = sentVariables();
    expect(operationName).toBe("UpdateCustomer");
    expect(variables.id).toBe("cus_1");
    expect(variables.input).toMatchObject({
      code: "ACME",
      stateId: "us_oh",
      dotNumber: "1234567",
      mcNumber: null,
      brokerVettingEnabled: true,
      checkCallIntervalHours: 4,
      etaRiskThresholdMinutes: null,
      statusUpdatePreference: "Arrivals",
      statusUpdateRecipients: "ops@acme.test",
      version: 3,
      billingProfile: {
        creditLimit: "25000.5",
        lateChargeRate: "1.5",
        minConsolidatedAmount: null,
        revenueAccountId: "gla_rev",
        requirePONumber: true,
        documentTypeIds: ["dt_pod", "dt_bol"],
      },
      emailProfile: { toRecipients: "ap@acme.test", includeShipmentDetail: true },
    });
    const billingProfile = (variables.input as { billingProfile: Record<string, unknown> })
      .billingProfile;
    expect(billingProfile).not.toHaveProperty("creditBalance");
    expect(billingProfile).not.toHaveProperty("creditHoldAutomatic");
    expect(billingProfile).not.toHaveProperty("documentTypes");
    expect(result).toEqual({ id: "cus_1", version: 4 });
  });

  it("creates a customer without profiles when the form holds none", async () => {
    requestGraphQL.mockResolvedValue({ createCustomer: { id: "cus_2", version: 1 } });

    await new CustomerService().create({
      ...customer,
      billingProfile: undefined,
      emailProfile: undefined,
    });

    const { operationName, variables } = sentVariables();
    expect(operationName).toBe("CreateCustomer");
    expect(variables.input).toMatchObject({ billingProfile: null, emailProfile: null });
  });

  it("sends only the status when a row's status changes", async () => {
    requestGraphQL.mockResolvedValue({ patchCustomer: { id: "cus_1", status: "Inactive" } });

    await new CustomerService().patch("cus_1", { status: "Inactive" });

    expect(sentVariables()).toMatchObject({
      operationName: "PatchCustomer",
      variables: { id: "cus_1", input: { status: "Inactive" } },
    });
  });

  it("bulk updates status by id", async () => {
    requestGraphQL.mockResolvedValue({ bulkUpdateCustomerStatus: [] });

    await new CustomerService().bulkUpdateStatus({ customerIds: ["cus_1"], status: "Inactive" });

    expect(sentVariables()).toMatchObject({
      operationName: "BulkUpdateCustomerStatus",
      variables: { input: { customerIds: ["cus_1"], status: "Inactive" } },
    });
  });
});

describe("AccessorialChargeService", () => {
  beforeEach(() => requestGraphQL.mockReset());

  it("creates a charge with its amount as a decimal string", async () => {
    requestGraphQL.mockResolvedValue({ createAccessorialCharge: { id: "acc_1", version: 1 } });

    await new AccessorialChargeService().create({
      status: "Active",
      code: "DET",
      description: "Detention",
      method: "PerUnit",
      rateUnit: "Hour",
      amount: 75.25,
    } as AccessorialCharge);

    expect(sentVariables()).toEqual({
      document: expect.anything(),
      operationName: "CreateAccessorialCharge",
      variables: {
        input: {
          status: "Active",
          code: "DET",
          description: "Detention",
          method: "PerUnit",
          rateUnit: "Hour",
          amount: "75.25",
          version: undefined,
        },
      },
    });
  });
});

describe("FleetCodeService", () => {
  beforeEach(() => requestGraphQL.mockReset());

  it("keeps the mileage goal the form does not show", async () => {
    requestGraphQL.mockResolvedValue({ updateFleetCode: { id: "fc_1", version: 3 } });

    await new FleetCodeService().update("fc_1", {
      status: "Active",
      code: "EAST",
      description: "",
      managerId: "usr_1",
      revenueGoal: 1000,
      deadheadGoal: null,
      mileageGoal: 9000,
      color: "",
      version: 2,
    } as FleetCode);

    expect(sentVariables()).toMatchObject({
      operationName: "UpdateFleetCode",
      variables: {
        id: "fc_1",
        input: {
          code: "EAST",
          description: null,
          managerId: "usr_1",
          revenueGoal: 1000,
          deadheadGoal: null,
          mileageGoal: 9000,
          color: null,
          version: 2,
        },
      },
    });
  });

  it("bulk updates fleet code status", async () => {
    requestGraphQL.mockResolvedValue({ bulkUpdateFleetCodeStatus: [] });

    await new FleetCodeService().bulkUpdateStatus({ fleetCodeIds: ["fc_1"], status: "Inactive" });

    expect(sentVariables()).toMatchObject({
      operationName: "BulkUpdateFleetCodeStatus",
      variables: { input: { fleetCodeIds: ["fc_1"], status: "Inactive" } },
    });
  });
});

describe("HoldReasonService", () => {
  beforeEach(() => requestGraphQL.mockReset());

  it("keeps the external codes the form does not edit", async () => {
    requestGraphQL.mockResolvedValue({ updateHoldReason: { id: "hr_1", version: 2 } });

    await new HoldReasonService().update("hr_1", {
      type: "ComplianceHold",
      code: "MISSING_POD",
      label: "Missing POD",
      description: "",
      active: true,
      defaultSeverity: "Blocking",
      defaultBlocksDispatch: false,
      defaultBlocksDelivery: false,
      defaultBlocksBilling: true,
      defaultVisibleToCustomer: false,
      sortOrder: 20,
      externalMap: { edi: "POD" },
      version: 1,
    } as HoldReason);

    expect(sentVariables()).toMatchObject({
      operationName: "UpdateHoldReason",
      variables: {
        id: "hr_1",
        input: { code: "MISSING_POD", description: null, externalMap: { edi: "POD" }, version: 1 },
      },
    });
  });
});
