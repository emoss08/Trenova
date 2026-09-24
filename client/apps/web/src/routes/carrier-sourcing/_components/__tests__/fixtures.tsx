import { buildProfile } from "@/components/carrier-intelligence/__tests__/fixtures";
import type { SourcingCandidate } from "@/lib/carrier-sourcing";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { type ReactNode, useState } from "react";
import { MemoryRouter } from "react-router";

export function buildCandidate(overrides: Partial<SourcingCandidate> = {}): SourcingCandidate {
  return {
    dotNumber: "1234567",
    legalName: "Blue Ridge Freight LLC",
    existingCarrierId: null,
    riskLevel: "High",
    findings: [],
    profile: buildProfile(),
    provider: "CarrierOK",
    depth: null,
    depthFetchedAt: null,
    fetchedDepth: null,
    notFound: false,
    laneMatches: null,
    asOf: null,
    fetchedAt: null,
    confirmedAt: null,
    sourceAsOf: null,
    ...overrides,
  };
}

export function TestProviders({ children }: { children: ReactNode }) {
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter>
        <MemoryRouter>{children}</MemoryRouter>
      </NuqsTestingAdapter>
    </QueryClientProvider>
  );
}

export { buildFinding, buildProfile } from "@/components/carrier-intelligence/__tests__/fixtures";
