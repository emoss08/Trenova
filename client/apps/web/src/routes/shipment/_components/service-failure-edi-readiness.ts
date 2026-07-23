export function ediReadinessLabel(action: string, reason?: string) {
  if (action === "generated") return "Generated";
  if (action === "duplicate") return "Generated";
  if (action === "blocked") return "Blocked";
  if (reason === "ready" || reason === "ready_for_generation") return "Ready for generation";
  if (reason === "service failure 214 trigger disabled") return "Not configured";
  if (reason === "shipment customer is not linked to an EDI partner") return "No customer partner";
  if (reason === "no outbound EDI partner for shipment customer") return "No partner";
  if (reason === "EDI partner is inactive or outbound disabled") return "Partner inactive";
  if (reason === "service failure 214 partner document profile inactive") return "Profile inactive";
  if (reason === "shipment status capability disabled") return "Capability off";
  if (reason === "ambiguous service failure 214 partner document profile") return "Ambiguous";
  return "Skipped";
}
