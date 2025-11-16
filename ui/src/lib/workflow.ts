import { VariableCategory } from "@/types/workflow";

export const VARIABLE_CATEGORIES = [
  {
    label: "Trigger Data",
    description: "Data from the event that started this workflow",
    variables: [
      {
        value: "{{trigger.shipmentId}}",
        label: "Shipment ID",
        description: "ID of the shipment that triggered the workflow",
      },
      {
        value: "{{trigger.status}}",
        label: "Status",
        description: "Current status value from the trigger",
      },
      {
        value: "{{trigger.customerId}}",
        label: "Customer ID",
        description: "Associated customer ID",
      },
      {
        value: "{{trigger.proNumber}}",
        label: "PRO Number",
        description: "Shipment PRO number",
      },
    ],
  },
  {
    label: "Previous Node Output",
    description: "Results from the node that executed before this one",
    variables: [
      {
        value: "{{previousNode.result}}",
        label: "Result Data",
        description: "Full result object from previous action",
      },
      {
        value: "{{previousNode.success}}",
        label: "Success Status",
        description: "Whether the previous action succeeded (true/false)",
      },
      {
        value: "{{previousNode.message}}",
        label: "Message",
        description: "Message or error from previous action",
      },
    ],
  },
  {
    label: "Workflow Context",
    description: "Information about the current workflow execution",
    variables: [
      {
        value: "{{workflow.executionId}}",
        label: "Execution ID",
        description: "Unique ID for this workflow run",
      },
      {
        value: "{{workflow.startedAt}}",
        label: "Started At",
        description: "Timestamp when workflow started",
      },
      {
        value: "{{workflow.organizationId}}",
        label: "Organization ID",
        description: "ID of the organization running this workflow",
      },
    ],
  },
] satisfies VariableCategory[];
