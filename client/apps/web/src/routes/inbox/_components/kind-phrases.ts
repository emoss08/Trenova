import { defineLabels } from "@trenova/shared/i18n/labels";

/**
 * A kind of message as a sentence names it, article and all: "When an
 * invoice arrives…". Each phrase is whole so a language can inflect it; the
 * article is never chosen from the English spelling at runtime.
 */
export const KIND_WITH_ARTICLE = defineLabels({
  Tender: "a tender",
  RateConfirmation: "a rate confirmation",
  ProofOfDelivery: "a proof of delivery",
  Invoice: "an invoice",
  StatusRequest: "a status request",
  DetentionDispute: "a detention dispute",
  Other: "anything else",
});

/** A kind of message counted in passing: "11 of 11 rate cons". */
export const KIND_PLURAL = defineLabels({
  Tender: "tenders",
  RateConfirmation: "rate cons",
  ProofOfDelivery: "PODs",
  Invoice: "invoices",
  StatusRequest: "status requests",
  DetentionDispute: "detention disputes",
  Other: "other messages",
});
