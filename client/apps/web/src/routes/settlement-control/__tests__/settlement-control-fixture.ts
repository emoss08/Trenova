import type { SettlementControl } from "@/types/settlement-control";

/**
 * A settlement control as the `settlementControl` query sends it for a new organization: the
 * server's defaults (weekly, ending Saturday, paid five days later, pay earned on delivery, a
 * 25% net pay check over eight settlements, quarterly escrow interest), with its decimals as
 * the strings the `Decimal` scalar carries.
 */
export const savedSettlementControl: SettlementControl = {
  id: "stlc_01",
  organizationId: "org_01",
  businessUnitId: "bu_01",
  version: 2,
  payPeriodFrequency: "Weekly",
  periodEndDayOfWeek: 6,
  payDelayDays: 5,
  payTrigger: "ShipmentDelivered",
  autoGenerateBatches: false,
  autoAttachAccruals: false,
  varianceThresholdPct: "25",
  varianceLookbackWeeks: 8,
  autoApproveClean: false,
  requireSeparateApprover: true,
  autoPostOnApprove: false,
  allowNegativeNet: false,
  defaultEscrowInterestRate: "0",
  escrowInterestFrequencyMonths: 3,
};
