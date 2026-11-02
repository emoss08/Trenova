import {
  billingQueueItemSchema,
  type BillingQueueItem,
  type BillingQueueStatus,
} from "@trenova/shared/types/billing-queue";

/**
 * The shipment from the field report, as `GET /billing-queue/:id/` returns it:
 * $2,850 freight split by amount between Peak ($1,500) and Acme ($1,350), and a
 * $1,154.38 detention charge nobody split, so it bills whole to Acme.
 */
export const DETENTION_ID = "ac_det";

const freightLine = {
  kind: "Freight",
  additionalChargeId: "",
  description: "Freight",
  chargeTotal: "2850",
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
};

const detentionLine = {
  kind: "Accessorial",
  additionalChargeId: DETENTION_ID,
  description: "Detention Fee",
  chargeTotal: "1154.38",
  percent: "100",
  method: "",
  partial: false,
  payers: [
    {
      payerId: "cus_acme",
      payerName: "Acme Manufacturing",
      payerCode: "ACME",
      amount: "1154.38",
      percent: "100",
    },
  ],
};

const payers = [
  { id: "cus_acme", name: "Acme Manufacturing", code: "ACME" },
  { id: "cus_peak", name: "Peak Distributing", code: "PEAK" },
];

export const ACME_SHARE_JSON = {
  payerId: "cus_acme",
  isSplit: true,
  payers,
  lines: [
    { ...freightLine, amount: "1350", percent: "47.368421" },
    { ...detentionLine, amount: "1154.38" },
  ],
  otherPayerLines: [],
  freightAmount: "1350",
  accessorialAmount: "1154.38",
  totalAmount: "2504.38",
  shipmentTotal: "4004.38",
  resolutionError: "",
};

export const PEAK_SHARE_JSON = {
  payerId: "cus_peak",
  isSplit: true,
  payers,
  lines: [{ ...freightLine, amount: "1500", percent: "52.631579" }],
  otherPayerLines: [{ ...detentionLine, amount: "0", percent: null }],
  freightAmount: "1500",
  accessorialAmount: "0",
  totalAmount: "1500",
  shipmentTotal: "4004.38",
  resolutionError: "",
};

const shipment = {
  id: "shp_1",
  proNumber: "SEED-DET-001",
  customerId: "cus_acme",
  customer: {
    id: "cus_acme",
    status: "Active",
    code: "ACME",
    name: "Acme Manufacturing",
    addressLine1: "4100 Industrial Pkwy",
    city: "Toledo",
    stateId: "st_oh",
    postalCode: "43612",
  },
  billToCustomerId: null,
  freightChargeAmount: 2850,
  tariffInputs: {},
  freightLines: [],
  otherChargeAmount: 1154.38,
  totalChargeAmount: 4004.38,
  formulaTemplateId: null,
  formulaTemplate: null,
  additionalCharges: [
    {
      id: DETENTION_ID,
      accessorialChargeId: "acc_det",
      method: "Flat",
      amount: 1154.38,
      unit: 1,
      isSystemGenerated: false,
      accessorialCharge: { id: "acc_det", code: "DET", description: "Detention Fee" },
    },
  ],
  chargeAllocations: [
    {
      id: "chal_peak",
      chargeKind: "Freight",
      shipmentId: "shp_1",
      billToCustomerId: "cus_peak",
      method: "Amount",
      percent: null,
      amount: 1500,
      sequence: 0,
      version: 0,
    },
    {
      id: "chal_acme",
      chargeKind: "Freight",
      shipmentId: "shp_1",
      billToCustomerId: "cus_acme",
      method: "Amount",
      percent: null,
      amount: 1350,
      sequence: 1,
      version: 0,
    },
  ],
};

/**
 * A billing queue item as the server sends it, parsed by the schema every response goes
 * through, so a fixture the contract would refuse fails here rather than in a component.
 */
export function queueItem({
  payer,
  status = "InReview",
  share,
  shipment: shipmentJson = shipment,
  allocatedTotalAmount,
}: {
  payer: "acme" | "peak";
  status?: BillingQueueStatus;
  share?: unknown;
  shipment?: unknown;
  allocatedTotalAmount?: number;
}): BillingQueueItem {
  const isAcme = payer === "acme";
  return billingQueueItemSchema.parse({
    id: isAcme ? "bqi_acme" : "bqi_peak",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: "shp_1",
    billToCustomerId: isAcme ? "cus_acme" : "cus_peak",
    allocatedTotalAmount: allocatedTotalAmount ?? (isAcme ? 2504.38 : 1500),
    number: isAcme ? "INV2609000022" : "INV2609000023",
    status,
    billType: "Invoice",
    isAdjustmentOrigin: false,
    requiresReplacementReview: false,
    adjustmentContext: {},
    version: 1,
    createdAt: 1_789_000_000,
    updatedAt: 1_789_000_000,
    shipment: shipmentJson,
    billToCustomer: isAcme
      ? { id: "cus_acme", name: "Acme Manufacturing", code: "ACME" }
      : { id: "cus_peak", name: "Peak Distributing", code: "PEAK" },
    payerShare: share ?? (isAcme ? ACME_SHARE_JSON : PEAK_SHARE_JSON),
    detentionHolds: [],
  });
}

/** Acme's item as the save answers once the detention line is removed: freight alone. */
export function acmeItemWithoutDetention(): BillingQueueItem {
  return queueItem({
    payer: "acme",
    allocatedTotalAmount: 1350,
    share: {
      ...ACME_SHARE_JSON,
      lines: [ACME_SHARE_JSON.lines[0]],
      accessorialAmount: "0",
      totalAmount: "1350",
      shipmentTotal: "2850",
    },
    shipment: {
      ...shipment,
      additionalCharges: [],
      otherChargeAmount: 0,
      totalChargeAmount: 2850,
    },
  });
}

export const LUMPER_ID = "ac_lump";

/** Acme's item as the save answers once a $185 lumper is added, billed whole to Acme. */
export function acmeItemWithLumper(): BillingQueueItem {
  return queueItem({
    payer: "acme",
    allocatedTotalAmount: 2689.38,
    share: {
      ...ACME_SHARE_JSON,
      lines: [
        ...ACME_SHARE_JSON.lines,
        {
          ...detentionLine,
          additionalChargeId: LUMPER_ID,
          description: "Lumper",
          chargeTotal: "185",
          amount: "185",
          payers: [{ ...detentionLine.payers[0]!, amount: "185" }],
        },
      ],
      accessorialAmount: "1339.38",
      totalAmount: "2689.38",
      shipmentTotal: "4189.38",
    },
    shipment: {
      ...shipment,
      additionalCharges: [
        ...shipment.additionalCharges,
        {
          id: LUMPER_ID,
          accessorialChargeId: "acc_lump",
          method: "Flat",
          amount: 185,
          unit: 1,
          isSystemGenerated: false,
          accessorialCharge: { id: "acc_lump", code: "LUMP", description: "Lumper" },
        },
      ],
      otherChargeAmount: 1339.38,
      totalChargeAmount: 4189.38,
    },
  });
}
