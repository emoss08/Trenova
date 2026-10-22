import { formulaCatalog } from "@trenova/shared/rules/formula";
import { rulesSync } from "@trenova/shared/rules/runtime";
import {
  formulaSchemaResponseSchema,
  type VariableDefinitionInput,
} from "@trenova/shared/types/formula-template";
import { buildKnownIdentifiers, type KnownIdentifiers } from "../known-identifiers";

/**
 * A shipment schema in the shape `GET /formula-templates/schema` answers
 * (docs/engineering/rules-engine.md, "The formula server API"): the record's
 * fields with their type, description, category and whether they can be
 * empty, an enum, and a list's item fields named `list[].field`.
 */
export const shipmentSchema = formulaSchemaResponseSchema.parse({
  schemaId: "shipment",
  variables: [
    { name: "proNumber", type: "string", description: "PRO number", category: "shipment" },
    { name: "baseRate", type: "number", description: "Base rate", category: "shipment" },
    { name: "totalDistance", type: "number", description: "Miles", category: "shipment" },
    { name: "totalWeight", type: "number", description: "Pounds", category: "shipment" },
    {
      name: "weight",
      type: "number",
      description: "Declared weight",
      category: "shipment",
      nullable: true,
    },
    { name: "hasHazmat", type: "boolean", description: "Carries hazmat", category: "shipment" },
    {
      name: "status",
      type: "string",
      description: "Status",
      category: "shipment",
      enum: ["New", "InTransit"],
    },
    { name: "pickupDate", type: "date", description: "Pickup", category: "shipment" },
    { name: "customer", type: "object", description: "Customer", category: "customer" },
    { name: "customer.name", type: "string", description: "Customer name", category: "customer" },
    { name: "origin.state", type: "string", description: "Origin state", category: "origin" },
    { name: "origin.zip", type: "string", description: "Origin ZIP", category: "origin" },
    {
      name: "destination.state",
      type: "string",
      description: "Destination state",
      category: "destination",
    },
    { name: "stops", type: "array", description: "Stops", category: "shipment" },
    { name: "stops[].type", type: "string", description: "Stop type", category: "shipment" },
    {
      name: "stops[].weight",
      type: "number",
      description: "Weight at the stop",
      category: "shipment",
      nullable: true,
    },
  ],
});

/** What the editor knows with the shipment schema and these inputs, on the loaded module. */
export function knownFor(inputs: VariableDefinitionInput[] = []): KnownIdentifiers {
  return buildKnownIdentifiers(formulaCatalog(rulesSync()), shipmentSchema, inputs);
}
