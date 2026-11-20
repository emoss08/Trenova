import { defineLabels } from "@trenova/shared/i18n/labels";

/** What woke a flow instance: the kinds the server journals (pkg/lading flow.go, edilading, ediflowclock). */
export const FLOW_EVENT_LABELS = defineLabels({
  received: "Received from the partner",
  trenova: "Happened in Trenova",
  planner: "Planner decided",
  sent: "Sent to the partner",
  ack: "Acknowledgment received",
  timer: "Deadline reached",
  load: "Load changed",
  released: "Released",
  advanced: "Clock advanced",
});

/** What a flow asked Trenova to do in answer to an event. */
export const FLOW_ACTION_LABELS = defineLabels({
  acknowledge: "Acknowledge",
  create: "Create the load",
  update: "Update the load",
  cancel: "Cancel the load",
  apply: "Apply",
  hold: "Hold",
  ignore: "Ignore",
  askPlanner: "Ask a planner",
  send: "Send",
  resend: "Resend",
  escalate: "Escalate",
});

/** Where a flow sees its load stand. */
export const FLOW_LOAD_STATE_LABELS = defineLabels({
  open: "Open",
  dispatched: "Dispatched",
  delivered: "Delivered",
  invoiced: "Invoiced",
  cancelled: "Cancelled",
});

/** The label of a server value, or the fallback for one this client does not know yet. */
export function flowLabel(
  labels: Readonly<Record<string, string>>,
  value: string,
  fallback: string,
): string {
  return Object.hasOwn(labels, value) ? labels[value] : fallback;
}
