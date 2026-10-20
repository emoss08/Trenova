// A plan the Dispatch agent proposed to recover a load the receiver's dock
// turned away: move the delivery, swap the driver, bill the wait, tell the
// customer. Four steps over three records, written from the server's
// contract: AssistantPlan and AssistantProposal (REST) and the plan preview
// (GraphQL AgentPlanPreviewFields). Every time is relative to the anchor.

export const DESK_PLAN_THREAD_ID = "athr_desk_plan";
export const DESK_PLAN_ID = "apl_tyson_recover";
export const DESK_PLAN_LATER_ID = "apl_tyson_finish";
export const DESK_PLAN_AGENT_ID = "agdef_dispatch";
export const DESK_PLAN_ARTIFACT_ID = "aart_plan_tyson";
export const DESK_PLAN_SLUG = "recover-s-240201";
export const DESK_PLAN_TITLE = "Recover S-240201 after the Tyson dock delay";
export const DESK_PLAN_SUMMARY =
  "Tyson's dock turned the truck away at 2:00 PM; this moves the delivery, swaps the driver, bills the wait and tells the customer.";
export const DESK_PLAN_DIGEST = "sha256:7f3a91c04be2d8e5a6f0c21e";

const SHIPMENT = { entityType: "shipment", id: "shp_240201" };
const DRIVER_ROSA = { entityType: "worker", id: "wrk_rosa" };
const DRIVER_SAMUEL = { entityType: "worker", id: "wrk_samuel" };
const CUSTOMER = { entityType: "customer", id: "cus_tyson" };

export const DESK_PLAN_STEPS = [
  {
    id: "aprop_tyson_1",
    toolName: "stop_update_appointment",
    rationale: "Move the Tyson delivery to tomorrow, 8:00 AM",
  },
  {
    id: "aprop_tyson_2",
    toolName: "dispatch_assign_worker",
    rationale: "Swap Rosa for Samuel, who has the hours for an 8 AM delivery",
  },
  {
    id: "aprop_tyson_3",
    toolName: "shipment_add_charge",
    rationale: "Bill the 4 hours of detention at the dock",
  },
  {
    id: "aprop_tyson_4",
    toolName: "email_send",
    rationale: "Tell Tyson's receiving desk about the new time",
  },
];

function field(path, label, before, after, extra = {}) {
  return {
    path,
    label,
    valueType: "text",
    before,
    after,
    beforeRef: null,
    afterRef: null,
    withheld: false,
    volatile: false,
    truncated: false,
    changedSinceProposed: false,
    proposedBefore: null,
    projectedFromStep: 0,
    ...extra,
  };
}

function ref(record, label) {
  return { resource: record.entityType, id: record.id, label, record, withheld: false };
}

function change(record, label, operation, fields, extra = {}) {
  return {
    resource: record.entityType,
    record,
    entityId: record.id,
    label,
    operation,
    version: 12,
    withheld: false,
    dependsOnStep: 0,
    fields,
    omittedFields: 0,
    message: null,
    money: null,
    ...extra,
  };
}

/** What each step changes, as the server previews it. */
export function deskPlanStepChanges() {
  return [
    {
      summary: "Would move stop 2 of S-240201 to Oct 21, 8:00 AM.",
      changes: [
        change(SHIPMENT, "S-240201", "Update", [
          field(
            "stops[1].appointmentStart",
            "Stop 2 appointment",
            "Oct 20, 2:00 PM",
            "Oct 21, 8:00 AM",
          ),
          field(
            "stops[1].appointmentWindow",
            "Appointment window",
            "2:00 – 4:00 PM",
            "8:00 – 10:00 AM",
          ),
          field("stops[1].confirmationNumber", "Confirmation #", "TY-55102", "TY-55187"),
        ]),
      ],
      warnings: [],
    },
    {
      summary: "Would put Samuel Okafor on move 1 of S-240201 in Rosa Diaz's place.",
      changes: [
        change(
          { entityType: "shipment", id: "shp_240201" },
          "Move 1 of S-240201",
          "Update",
          [
            field("primaryWorkerId", "Primary driver", "wrk_rosa", "wrk_samuel", {
              beforeRef: ref(DRIVER_ROSA, "Rosa Diaz"),
              afterRef: ref(DRIVER_SAMUEL, "Samuel Okafor"),
            }),
          ],
          { entityId: "smv_240201_1" },
        ),
      ],
      warnings: [],
    },
    {
      summary: "Would add $150.00 of detention to S-240201.",
      changes: [
        change(
          SHIPMENT,
          "S-240201",
          "Update",
          [field("detention", "Detention", "$0.00", "$150.00", { projectedFromStep: 1 })],
          {
            dependsOnStep: 1,
            money: {
              currency: "USD",
              lines: [
                { label: "Linehaul", before: "2340.00", after: "2340.00" },
                { label: "Detention", before: "0.00", after: "150.00" },
              ],
              totalBefore: "2340.00",
              totalAfter: "2490.00",
              delta: "150.00",
              withheld: false,
            },
          },
        ),
      ],
      warnings: [
        {
          code: "depends_on_step",
          args: ["1"],
          message: "Step 1 changes this record first.",
          reasons: [],
        },
      ],
    },
    {
      summary: "Would email Tyson Foods' receiving desk about the new delivery time.",
      changes: [
        change(CUSTOMER, "Tyson Foods", "Send", [], {
          version: null,
          message: {
            channel: "Email",
            from: "dispatch@trenova.example",
            to: ["receiving.springdale@tyson.example"],
            cc: ["mgarza@tyson.example"],
            bcc: [],
            attachments: [],
            subject: "S-240201 delivery moved to Oct 21, 8:00 AM",
            body: "Hello,\n\nThe load for PO 4471-88 will arrive tomorrow, Oct 21, at 8:00 AM under confirmation TY-55187, instead of today at 2:00 PM. Driver Samuel Okafor, tractor T-101.\n\nThank you,\nDispatch",
            bodyTruncated: false,
            visibility: "",
            cadence: "",
            templateVersionId: null,
          },
        }),
      ],
      warnings: [
        {
          code: "already_told_customer",
          args: [],
          message: "The customer has already been told about this.",
          reasons: [],
        },
      ],
    },
  ];
}
