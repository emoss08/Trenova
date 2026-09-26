import type {
  PlanPreview,
  PreviewFieldChange,
  PreviewMessage,
  PreviewMoney,
  PreviewReason,
  PreviewRecordChange,
  PreviewWarning,
  ProposalPreview,
} from "@/lib/graphql/agent-preview";

/*
Fixtures written from the contract (services/tms/internal/api/graphql/schema/
agentpreview.graphqls), not from the components: every field the SDL declares
is present, nullable ones default to null the way the server sends an absent
value, and each builder takes overrides so a test can construct what the
contract permits and no other fixture does — a withheld field with no values,
a moved value, a Run record with no resource, a money block with no before.
*/

export function reason(overrides: Partial<PreviewReason> = {}): PreviewReason {
  return {
    field: "bol",
    label: "BOL",
    message: "BOL is already in use by shipment SEED-DET-009",
    param: "shipment.bol",
    ...overrides,
  };
}

export function warning(overrides: Partial<PreviewWarning> = {}): PreviewWarning {
  return {
    code: "would_fail",
    args: [],
    message: "This would be refused as it stands: validation failed",
    reasons: [],
    ...overrides,
  };
}

export function field(overrides: Partial<PreviewFieldChange> = {}): PreviewFieldChange {
  return {
    path: "status",
    label: "Status",
    valueType: "text",
    before: "Old value",
    after: "New value",
    beforeRef: null,
    afterRef: null,
    withheld: false,
    volatile: false,
    truncated: false,
    changedSinceProposed: false,
    proposedBefore: null,
    projectedFromStep: 0,
    ...overrides,
  };
}

export function message(overrides: Partial<PreviewMessage> = {}): PreviewMessage {
  return {
    channel: "Email",
    from: "dispatch@trenova.test",
    to: ["ap@customer.test"],
    cc: [],
    bcc: [],
    attachments: [],
    subject: "Delivery update",
    body: "Your load delivered at 14:02.",
    bodyTruncated: false,
    visibility: "",
    cadence: "",
    templateVersionId: null,
    ...overrides,
  };
}

export function money(overrides: Partial<PreviewMoney> = {}): PreviewMoney {
  return {
    currency: "USD",
    lines: [{ label: "Detention", before: "0", after: "150.00" }],
    totalBefore: "1200.00",
    totalAfter: "1350.00",
    delta: "150.00",
    withheld: false,
    ...overrides,
  };
}

export function record(overrides: Partial<PreviewRecordChange> = {}): PreviewRecordChange {
  return {
    resource: "shipment",
    record: { entityType: "shipment", id: "shp_1" },
    entityId: "shp_1",
    label: "S-1001",
    operation: "Update",
    version: 4,
    withheld: false,
    dependsOnStep: 0,
    fields: [field()],
    omittedFields: 0,
    message: null,
    money: null,
    ...overrides,
  };
}

export function preview(overrides: Partial<ProposalPreview> = {}): ProposalPreview {
  return {
    proposalId: "aprop_1",
    tool: "update_shipment",
    summary: "Would change S-1001.",
    coverage: "Full",
    changes: [record()],
    warnings: [],
    stale: false,
    staleness: { pinned: true, proposedVersion: 4, currentVersion: 4, missing: false },
    targetVersion: 4,
    withheldCount: 0,
    omittedRecords: 0,
    recorded: false,
    computedAt: 1_790_000_000,
    digest: "sha256:aaaa",
    ...overrides,
  };
}

export function planPreview(overrides: Partial<PlanPreview> = {}): PlanPreview {
  return {
    planId: "apl_1",
    digest: "sha256:plan",
    stale: false,
    withheldCount: 0,
    computedAt: 1_790_000_000,
    steps: [
      { proposalId: "aprop_1", step: 1, preview: preview({ proposalId: "aprop_1" }) },
      {
        proposalId: "aprop_2",
        step: 2,
        preview: preview({
          proposalId: "aprop_2",
          digest: "sha256:bbbb",
          changes: [record({ dependsOnStep: 1, fields: [field({ projectedFromStep: 1 })] })],
          warnings: [
            warning({
              code: "depends_on_step",
              args: ["1"],
              message: "This step changes a record step 1 changes first.",
            }),
          ],
        }),
      },
    ],
    ...overrides,
  };
}
