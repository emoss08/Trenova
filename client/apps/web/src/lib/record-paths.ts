/**
 * Where a record opens. Each list page opens its edit panel from the address,
 * so a link can land a person on the record itself rather than on the list.
 */
function recordPanelPath(listPath: string, id: string): string {
  const params = new URLSearchParams({ panelType: "edit", panelEntityId: id });

  return `${listPath}?${params.toString()}`;
}

export function shipmentPath(id: string): string {
  return recordPanelPath("/shipment-management/shipments", id);
}

export function customerPath(id: string): string {
  return recordPanelPath("/billing/configuration-files/customers", id);
}

/** An agent run, opened in AI Control's activity list filtered to it. */
export function agentRunPath(id: string): string {
  const params = new URLSearchParams({
    tab: "activity",
    activity: "runs",
    fieldFilters: JSON.stringify([{ field: "id", operator: "eq", value: id }]),
  });

  return `/admin/agent-control?${params.toString()}`;
}
