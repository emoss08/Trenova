import { translate } from "@trenova/shared/i18n/runtime";

/** The badge for a service failure's 214, from what the server's preview decided and why. */
export function ediReadinessLabel(action: string, reason?: string) {
  if (action === "generated") return translate("Generated");
  if (action === "duplicate") return translate("Generated");
  if (action === "queued") return translate("Queued");
  if (action === "blocked") return translate("Blocked");
  if (reason === "ready" || reason === "ready_for_generation") {
    return translate("Ready for generation");
  }
  if (reason === "the partner's flow sends it with its next batch") return translate("Queued");
  if (reason === "the partner's flow sends no 214 on a service failure") {
    return translate("Not in the partner's flow");
  }
  if (reason === "the partner has no active Lading version") return translate("No active version");
  if (reason === "the partner's 214 map or rules refuse this service failure") {
    return translate("Refused by the partner's rules");
  }
  if (reason === "shipment customer is not linked to an EDI partner") {
    return translate("No customer partner");
  }
  if (reason === "no outbound EDI partner for shipment customer") return translate("No partner");
  if (reason === "EDI partner is inactive or outbound disabled") {
    return translate("Partner inactive");
  }
  if (reason === "shipment status capability disabled") return translate("Capability off");
  return translate("Skipped");
}
