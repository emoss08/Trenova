import { recordPath } from "@/config/record-links";

export function shipmentPath(id: string): string {
  return recordPath("shipment", id);
}

export function customerPath(id: string): string {
  return recordPath("customer", id);
}

/** An agent run, opened in AI Control's activity list filtered to it. */
export function agentRunPath(id: string): string {
  return recordPath("agent_run", id);
}
