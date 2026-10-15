import { defineLabels } from "@trenova/shared/i18n/labels";

/** A kind of message counted in passing: "11 of 11 rate cons". */
export const KIND_PLURAL = defineLabels({
  Tender: "tenders",
  RateConfirmation: "rate cons",
  ProofOfDelivery: "PODs",
  Invoice: "invoices",
  StatusRequest: "status requests",
  DetentionDispute: "detention disputes",
  FactorRemittance: "factor remittances",
  Other: "other messages",
});
