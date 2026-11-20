import type { CarrierSettlementControl } from "@/types/carrier-settlement-control";

/**
 * A saved carrier settlement control as the `carrierSettlementControl` query sends it: weekly
 * periods ending Saturday, paid five days later, cost counted on delivery, a $5.00 tolerance,
 * no automatic matching and no posting accounts chosen.
 */
export const savedCarrierSettlementControl: CarrierSettlementControl = {
  id: "carstlc_01",
  organizationId: "org_01",
  businessUnitId: "bu_01",
  version: 3,
  payPeriodFrequency: "Weekly",
  periodEndDayOfWeek: 6,
  payDelayDays: 5,
  autoGenerateBatches: false,
  payTrigger: "ShipmentDelivered",
  varianceToleranceMinor: 500,
  autoMatchInboundInvoices: false,
  autoAcceptWithinTolerance: false,
  holdUntilInvoiceMatched: false,
  requireSeparateApprover: true,
  autoPostOnApprove: false,
  defaultApAccountId: null,
  defaultPurchasedTransportationAccountId: null,
};
