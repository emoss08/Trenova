export const ediTransferEndpoints = {
  loadTenders: "/edi/load-tenders/",
  transfers: "/edi/transfers/",
} as const;

export function ediTransferListEndpoint(direction: "inbound" | "outbound", query: string) {
  if (!query) return `${ediTransferEndpoints.transfers}?direction=${direction}`;

  const normalizedQuery = query.startsWith("?") ? query : `?${query}`;
  const separator = normalizedQuery === "?" ? "" : "&";
  return `${ediTransferEndpoints.transfers}${normalizedQuery}${separator}direction=${direction}`;
}
