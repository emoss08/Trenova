import type { CarrierIntelFinding, CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";

export function emptyProfile(overrides: Partial<CarrierIntelProfile> = {}): CarrierIntelProfile {
  return {
    coverage: [],
    identity: null,
    authority: null,
    insurance: null,
    safety: null,
    basics: null,
    inspections: null,
    crashes: null,
    fleet: null,
    equipment: null,
    contacts: null,
    operations: null,
    changeHistory: null,
    network: null,
    lanes: null,
    benchmarks: null,
    ...overrides,
  };
}

export function buildProfile(overrides: Partial<CarrierIntelProfile> = {}): CarrierIntelProfile {
  return {
    coverage: ["Identity", "Authority", "Insurance", "Fleet"],
    identity: {
      dotNumber: "1234567",
      docketPrefix: "MC",
      docketNumber: "765432",
      legalName: "Blue Ridge Freight LLC",
      dbaName: null,
      ein: null,
      usdotStatus: "Active",
      entityType: "Carrier",
      carrierOperation: "Interstate",
      dotAddedAt: null,
      dotAgeDays: 2400,
      physicalAddress: {
        line1: "1 Main St",
        city: "Asheville",
        state: "NC",
        postalCode: "28801",
        country: "US",
        undelivered: false,
      },
      mailingAddress: null,
    },
    authority: {
      common: {
        status: "Active",
        pending: false,
        underReview: false,
        revocationPending: false,
        grantedAt: null,
        ageDays: 6570,
      },
      contract: null,
      broker: null,
      totalRevocations: 0,
      lastRevocationAt: null,
      history: null,
    },
    insurance: null,
    safety: null,
    basics: null,
    inspections: null,
    crashes: null,
    fleet: {
      powerUnits: 101844,
      drivers: 50,
      cdlDrivers: 50,
      ownedTractors: null,
      termLeasedTractors: null,
      ownedTrailers: null,
      termLeasedTrailers: null,
      trailers: null,
      trucks: null,
    },
    equipment: null,
    contacts: null,
    operations: null,
    changeHistory: null,
    network: null,
    lanes: null,
    benchmarks: null,
    ...overrides,
  };
}

export function buildFinding(overrides: Partial<CarrierIntelFinding> = {}): CarrierIntelFinding {
  return {
    code: "authority.too_new",
    category: "Authority",
    action: "Block",
    severity: "High",
    message: "Operating authority is younger than 180 days",
    unverifiable: false,
    unconfirmed: false,
    overridden: false,
    overrideId: null,
    overrideExpiresAt: null,
    ...overrides,
  };
}

export function IntelTestProviders({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}
