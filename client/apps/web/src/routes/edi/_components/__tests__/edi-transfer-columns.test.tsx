import { cleanup, render, screen } from "@testing-library/react";
import { translate } from "@trenova/shared/i18n/runtime";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { EDITransferRow } from "@/lib/graphql/edi-table";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { getTransferColumns } from "../edi-transfer-columns";

afterEach(cleanup);

type Meta = { apiField?: string };

// Partners are kept per organization: each side holds its own record of the
// other, and row-level security hides the other side's. A tender between two
// organizations on Trenova reaches each reader with only its own partner.
const internal = {
  id: "edilt_s2610_1366",
  sourceOrganizationId: "org_lonestar",
  sourceBusinessUnitId: "bu_lonestar",
  targetOrganizationId: "org_acme",
  targetBusinessUnitId: "bu_acme",
  sourcePartnerId: "edip_lonestar_acme",
  targetPartnerId: "edip_acme_lonestar",
  sourceShipmentId: "shp_lonestar_1366",
  targetShipmentId: null,
  inboundMessageId: null,
  status: "PendingApproval",
  tenderPayload: { bol: "BOL-55120" },
  mappingSnapshot: {},
  rejectionReason: null,
  failureReason: null,
  submittedAt: 1_791_716_400,
  processedAt: null,
  version: 1,
  createdAt: 1_791_716_400,
  updatedAt: 1_791_716_400,
  sourcePartner: null,
  targetPartner: null,
} as unknown as EDITransferRow;

const asReceiver: EDITransferRow = {
  ...internal,
  targetPartner: { id: "edip_acme_lonestar", code: "LONESTAR", name: "Lone Star Logistics" },
};

const asSender: EDITransferRow = {
  ...internal,
  sourcePartner: { id: "edip_lonestar_acme", code: "ACME", name: "Acme Foods" },
};

// A tender read from a partner's X12 file names the one partner on both sides.
const fromFile: EDITransferRow = {
  ...internal,
  sourceOrganizationId: "org_acme",
  sourceBusinessUnitId: "bu_acme",
  sourcePartnerId: "edip_walmart",
  targetPartnerId: "edip_walmart",
  sourceShipmentId: null,
  inboundMessageId: "edimsg_walmart_0001",
  sourcePartner: { id: "edip_walmart", code: "WALMART", name: "Walmart" },
  targetPartner: { id: "edip_walmart", code: "WALMART", name: "Walmart" },
};

function partnerColumn(direction: "inbound" | "outbound"): ColumnDef<EDITransferRow> {
  const column = getTransferColumns(direction, translate).find(
    (candidate) => candidate.id === "partner",
  );
  if (!column) {
    throw new Error("no partner column");
  }
  return column;
}

function drawPartner(direction: "inbound" | "outbound", row: EDITransferRow) {
  const cell = partnerColumn(direction).cell as (context: {
    row: { original: EDITransferRow };
  }) => ReactNode;
  render(<>{cell({ row: { original: row } })}</>);
}

describe("EDI transfer table partner column", () => {
  it("names who tendered an inbound load as the receiver knows them", () => {
    drawPartner("inbound", asReceiver);

    expect(screen.getByText("Lone Star Logistics")).toBeInTheDocument();
  });

  it("names who an outbound load went to as the sender knows them", () => {
    drawPartner("outbound", asSender);

    expect(screen.getByText("Acme Foods")).toBeInTheDocument();
  });

  it("names the partner of a tender read from their file", () => {
    drawPartner("inbound", fromFile);

    expect(screen.getByText("Walmart")).toBeInTheDocument();
  });

  it("points each table at the partner its own organization holds", () => {
    expect((partnerColumn("inbound").meta as Meta).apiField).toBe("targetPartnerId");
    expect((partnerColumn("outbound").meta as Meta).apiField).toBe("sourcePartnerId");
  });
});
