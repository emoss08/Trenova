import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ShipmentPermitsTab from "../shipment-permits-tab";

const { getPermitAssessment, listPermitRequirements, listPermits, recordDialog } = vi.hoisted(
  () => ({
    getPermitAssessment: vi.fn(),
    listPermitRequirements: vi.fn(),
    listPermits: vi.fn(),
    recordDialog: vi.fn(),
  }),
);

vi.mock("@/services/shipment", () => ({
  shipmentService: { getPermitAssessment, listPermitRequirements, listPermits },
}));
vi.mock("../../permit-dialogs", () => ({
  PermitRecordDialog: (props: { open: boolean; requirement: { id: string } | null }) => {
    if (props.open) recordDialog(props.requirement?.id ?? null);
    return null;
  },
  PermitWaiveDialog: () => null,
}));

function jurisdiction(code: string, name: string, widthLimit: number, sequence: number) {
  return {
    stateId: `st_${code.toLowerCase()}`,
    stateCode: code,
    stateName: name,
    sequence,
    verificationState: "Verified",
    maxWidthFeet: widthLimit,
    maxHeightFeet: 13.5,
    maxLengthFeet: 75,
    maxWeightPounds: 80000,
    widthHeadroomFeet: widthLimit - 10.17,
    heightHeadroomFeet: 0.5,
    lengthHeadroomFeet: 22,
    weightHeadroomPounds: 2000,
    requiresPermit: widthLimit < 10.17,
    restrictions: [],
  };
}

function requirementFor(code: string, name: string, sequence: number, extra = {}) {
  return {
    id: `req_${code.toLowerCase()}`,
    shipmentId: "shp_1",
    stateId: `st_${code.toLowerCase()}`,
    status: "Open",
    routeSequence: sequence,
    exceedances: [{ trigger: "Width", limit: 8.5, actual: 10.17, overBy: 1.67, superload: false }],
    escorts: [],
    restrictions: [],
    leadTimeDays: 5,
    validityDays: 10,
    estimatedFee: "140",
    isSuperload: false,
    satisfiedByPermitId: null,
    waivedById: null,
    derivedAt: 1,
    provenance: {
      ruleId: `rule_${code}`,
      stateCode: code,
      stateName: name,
      verificationState: "Verified",
      maxWidthFeet: 8.5,
      maxHeightFeet: 13.5,
      maxLengthFeet: 75,
      maxWeightPounds: 80000,
    },
    ...extra,
  };
}

// A 10' 2" wide load through New Jersey (permit filed), Pennsylvania and Ohio (still to file).
function assessment(overrides: Record<string, unknown> = {}) {
  return {
    shipmentId: "shp_1",
    measurements: { widthFeet: 10.17, heightFeet: 13, lengthFeet: 53, weightPounds: 78000 },
    jurisdictions: [
      jurisdiction("NJ", "New Jersey", 8.5, 0),
      jurisdiction("PA", "Pennsylvania", 8.5, 1),
      jurisdiction("OH", "Ohio", 8.5, 2),
    ],
    requirements: [],
    open: [{}, {}],
    totalEscorts: 0,
    totalEstimatedFee: "420",
    maxLeadTimeDays: 5,
    earliestPickup: 1_793_500_000,
    routeResolved: true,
    feeIsBaseOnly: false,
    ...overrides,
  };
}

function Harness() {
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <ShipmentPermitsTab
        shipment={{ id: "shp_1", version: 3, moves: [] } as unknown as Shipment}
      />
    </QueryClientProvider>
  );
}

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  getPermitAssessment.mockResolvedValue(assessment());
  listPermitRequirements.mockResolvedValue([
    requirementFor("NJ", "New Jersey", 0, { status: "Satisfied", satisfiedByPermitId: "pmt_nj" }),
    requirementFor("PA", "Pennsylvania", 1),
    requirementFor("OH", "Ohio", 2),
  ]);
  listPermits.mockResolvedValue([
    {
      id: "pmt_nj",
      shipmentId: "shp_1",
      stateId: "st_nj",
      permitNumber: "NJ-48213",
      status: "Active",
      expiresAt: 1_793_900_000,
      state: { id: "st_nj", abbreviation: "NJ", name: "New Jersey" },
    },
  ]);
});

describe("ShipmentPermitsTab", () => {
  it("says how many permits are left to file and how far through the checklist it is", async () => {
    render(<Harness />);

    expect(await screen.findByRole("heading", { name: "2 permits to file" })).toBeInTheDocument();
    expect(screen.getByText("1 of 3 done")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "Permits done" })).toHaveAttribute(
      "aria-valuenow",
      "1",
    );
    expect(screen.getByText("About $420.00 in fees")).toBeInTheDocument();
  });

  it("shows the width over its tightest limit", async () => {
    const { container } = render(<Harness />);
    await screen.findByRole("heading", { name: "2 permits to file" });

    const width = container.querySelector('[data-dimension="width"]') as HTMLElement;
    expect(width).toHaveAttribute("data-exceeded", "true");
    expect(within(width).getByText(/over$/)).toBeInTheDocument();
  });

  it("lists the states still to file first, each with its button, then the permit filed", async () => {
    const { container } = render(<Harness />);
    await screen.findByRole("heading", { name: "2 permits to file" });

    const rows = [...container.querySelectorAll("[data-permit-phase]")] as HTMLElement[];
    expect(rows.map((row) => [row.dataset.state, row.dataset.permitPhase])).toEqual([
      ["PA", "todo"],
      ["OH", "todo"],
      ["NJ", "filed"],
    ]);
    expect(within(rows[0]!).getByRole("button", { name: "Record permit" })).toBeInTheDocument();
    expect(within(rows[2]!).getByText("NJ-48213")).toBeInTheDocument();
    expect(within(rows[2]!).queryByRole("button", { name: "Record permit" })).toBeNull();
  });

  it("opens the record dialog for the state it was chosen on", async () => {
    render(<Harness />);
    await screen.findByRole("heading", { name: "2 permits to file" });

    await userEvent.click(screen.getAllByRole("button", { name: "Record permit" })[1]!);

    expect(recordDialog).toHaveBeenLastCalledWith("req_oh");
  });

  it("says permits are in place once every state is resolved", async () => {
    listPermitRequirements.mockResolvedValue([
      requirementFor("PA", "Pennsylvania", 1, { status: "Waived", waiverReason: "Escorted move" }),
    ]);
    getPermitAssessment.mockResolvedValue(assessment({ open: [] }));
    render(<Harness />);

    expect(await screen.findByRole("heading", { name: "Permits in place" })).toBeInTheDocument();
    expect(screen.getByText("Waived: Escorted move")).toBeInTheDocument();
  });

  it("does not call an unchecked route legal", async () => {
    listPermitRequirements.mockResolvedValue([]);
    listPermits.mockResolvedValue([]);
    getPermitAssessment.mockResolvedValue(
      assessment({ routeResolved: false, jurisdictions: [], open: [], maxLeadTimeDays: 0 }),
    );
    render(<Harness />);

    expect(await screen.findByRole("heading", { name: "Route not checked" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Legal on this route" })).toBeNull();
  });

  it("asks for the cargo's dimensions before there is anything to assess", async () => {
    getPermitAssessment.mockResolvedValue(
      assessment({
        measurements: { widthFeet: 0, heightFeet: 0, lengthFeet: 0, weightPounds: 0 },
      }),
    );
    render(<Harness />);

    expect(
      await screen.findByText(/Add length, width and height to the commodity lines/),
    ).toBeInTheDocument();
  });

  it("offers a retry when the assessment cannot be loaded", async () => {
    getPermitAssessment.mockRejectedValueOnce(new Error("offline"));
    render(<Harness />);

    expect(
      await screen.findByText("The permit assessment could not be loaded"),
    ).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Retry" }));

    expect(await screen.findByRole("heading", { name: "2 permits to file" })).toBeInTheDocument();
  });
});
