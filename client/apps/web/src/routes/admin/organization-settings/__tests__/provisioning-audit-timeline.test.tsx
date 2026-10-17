import type { ProvisioningAuditRecord } from "@trenova/shared/types/iam";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { AuditTimeline } from "../_components/provisioning/audit-timeline";

/** A record as `GET …/iam/provisioning-audit` returns it. */
function record(overrides: Partial<ProvisioningAuditRecord>): ProvisioningAuditRecord {
  return {
    id: "par_01",
    organizationId: "org_01",
    directoryId: "scim_01",
    action: "create",
    resourceType: "User",
    externalId: "okta|00u1",
    resourceId: "usr_01",
    status: "succeeded",
    errorMessage: "",
    createdAt: 1_760_000_000,
    ...overrides,
  } as ProvisioningAuditRecord;
}

afterEach(cleanup);

describe("AuditTimeline", () => {
  it("names each event by what was done and to which kind of record", () => {
    render(
      <AuditTimeline
        isLoading={false}
        records={[
          record({ id: "par_1", action: "create" }),
          record({ id: "par_2", action: "update", resourceType: "Group" }),
          record({ id: "par_3", action: "deactivate" }),
          record({ id: "par_4", action: "delete", resourceType: "Group" }),
        ]}
      />,
    );

    expect(screen.getByText("Created · User")).toBeInTheDocument();
    expect(screen.getByText("Updated · Group")).toBeInTheDocument();
    expect(screen.getByText("Deactivated · User")).toBeInTheDocument();
    expect(screen.getByText("Deleted · Group")).toBeInTheDocument();
  });
});
