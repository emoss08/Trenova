export type EDIScriptPresetCategory =
  | "elementValue"
  | "repeatItem"
  | "condition"
  | "scriptLibrary";

export type EDIScriptPreset = {
  id: string;
  label: string;
  description: string;
  category: EDIScriptPresetCategory;
  code: string;
  recommendedFunctionName?: string;
};

const shipmentReferenceValueCode = `
def shipment_reference_value(ctx):
    shipment = ctx["shipment"]
    references = default(shipment.get("references"), [])
    for reference in references:
        value = coalesce(
            reference.get("value"),
            reference.get("referenceValue"),
            reference.get("referenceNumber"),
        )
        if exists(value):
            return value
    return coalesce(shipment.get("bol"), shipment.get("shipmentId"))
`;

const shipmentReferenceQualifierCode = `
def shipment_reference_qualifier(ctx):
    shipment = ctx["shipment"]
    references = default(shipment.get("references"), [])
    for reference in references:
        qualifier = coalesce(reference.get("qualifier"), reference.get("referenceQualifier"))
        if exists(qualifier):
            return qualifier
    return default(shipment.get("referenceQualifier"), "BM")
`;

const contactPhoneNormalizationCode = `
def normalized_contact_phone(ctx):
    partner = ctx["partner"]
    contact = default(partner.get("contact"), {})
    return normalize_phone(
        coalesce(
            contact.get("phone"),
            contact.get("phoneNumber"),
            ctx["shipment"].get("contactPhone"),
        )
    )
`;

const stopReasonCodeCode = `
def stop_reason_code(ctx, item):
    stop_type = item.get("type")
    if stop_type == "LD" or stop_type == "Pickup":
        return "LD"
    if stop_type == "UL" or stop_type == "Delivery":
        return "UL"
    return default(stop_type, "LD")
`;

const locationCodeFallbackCode = `
def location_code(ctx, item):
    return coalesce(
        item.get("locationCode"),
        item.get("code"),
        item.get("locationId"),
        item.get("locationName"),
    )
`;

const commodityDescriptionFallbackCode = `
def commodity_description(ctx, item):
    return default(
        coalesce(
            item.get("commodityDescription"),
            item.get("description"),
            item.get("name"),
        ),
        "Freight",
    )
`;

const accessorialCodeFallbackCode = `
def accessorial_code(ctx, item):
    return default(
        coalesce(
            item.get("accessorialCode"),
            item.get("code"),
            item.get("accessorialDescription"),
        ),
        "MSC",
    )
`;

const bolExistsInlineConditionCode = `
starlark:def include(ctx):
    return exists(ctx["shipment"].get("bol"))
`;

const pickupStopInlineConditionCode = `
starlark:def include(ctx, item):
    stop_type = item.get("type")
    return stop_type == "LD" or stop_type == "Pickup"
`;

const deliveryStopInlineConditionCode = `
starlark:def include(ctx, item):
    stop_type = item.get("type")
    return stop_type == "UL" or stop_type == "Delivery"
`;

const conditionLibraryCode = `
def include_bol_exists(ctx):
    return exists(ctx["shipment"].get("bol"))

def include_pickup_stop(ctx, item):
    stop_type = item.get("type")
    return stop_type == "LD" or stop_type == "Pickup"

def include_delivery_stop(ctx, item):
    stop_type = item.get("type")
    return stop_type == "UL" or stop_type == "Delivery"
`;

const valueLibraryCode = `
def first_shipment_reference_value(ctx):
    shipment = ctx["shipment"]
    references = default(shipment.get("references"), [])
    for reference in references:
        value = coalesce(
            reference.get("value"),
            reference.get("referenceValue"),
            reference.get("referenceNumber"),
        )
        if exists(value):
            return value
    return coalesce(shipment.get("bol"), shipment.get("shipmentId"))

def first_shipment_reference_qualifier(ctx):
    shipment = ctx["shipment"]
    references = default(shipment.get("references"), [])
    for reference in references:
        qualifier = coalesce(reference.get("qualifier"), reference.get("referenceQualifier"))
        if exists(qualifier):
            return qualifier
    return default(shipment.get("referenceQualifier"), "BM")

def normalized_partner_contact_phone(ctx):
    partner = ctx["partner"]
    contact = default(partner.get("contact"), {})
    return normalize_phone(
        coalesce(
            contact.get("phone"),
            contact.get("phoneNumber"),
            ctx["shipment"].get("contactPhone"),
        )
    )
`;

export const ediScriptPresets: EDIScriptPreset[] = [
  {
    id: "element.shipment_reference_value",
    label: "Shipment Reference Value",
    description: "Returns the first shipment reference value, falling back to BOL or shipment ID.",
    category: "elementValue",
    code: shipmentReferenceValueCode,
    recommendedFunctionName: "shipment_reference_value",
  },
  {
    id: "element.shipment_reference_qualifier",
    label: "Shipment Reference Qualifier",
    description: "Returns the first shipment reference qualifier, falling back to BM.",
    category: "elementValue",
    code: shipmentReferenceQualifierCode,
    recommendedFunctionName: "shipment_reference_qualifier",
  },
  {
    id: "element.contact_phone_normalization",
    label: "Contact Phone Normalization",
    description: "Normalizes partner contact phone with a shipment contact fallback.",
    category: "elementValue",
    code: contactPhoneNormalizationCode,
    recommendedFunctionName: "normalized_contact_phone",
  },
  {
    id: "repeat.stop_reason_code",
    label: "Stop Reason Code",
    description: "Maps pickup and delivery stop types to LD and UL.",
    category: "repeatItem",
    code: stopReasonCodeCode,
    recommendedFunctionName: "stop_reason_code",
  },
  {
    id: "repeat.location_code_fallback",
    label: "Location Code Fallback",
    description: "Uses location code, ID, or name from the current repeat item.",
    category: "repeatItem",
    code: locationCodeFallbackCode,
    recommendedFunctionName: "location_code",
  },
  {
    id: "repeat.commodity_description_fallback",
    label: "Commodity Description Fallback",
    description: "Uses commodity description, generic description, name, or Freight.",
    category: "repeatItem",
    code: commodityDescriptionFallbackCode,
    recommendedFunctionName: "commodity_description",
  },
  {
    id: "repeat.accessorial_code_fallback",
    label: "Accessorial Code Fallback",
    description: "Uses accessorial code, generic code, description, or MSC.",
    category: "repeatItem",
    code: accessorialCodeFallbackCode,
    recommendedFunctionName: "accessorial_code",
  },
  {
    id: "condition.bol_exists.reference",
    label: "BOL Exists Reference",
    description: "References include_bol_exists from a script library.",
    category: "condition",
    code: "starlark:include_bol_exists",
  },
  {
    id: "condition.bol_exists.inline",
    label: "BOL Exists Inline",
    description: "Defines an inline include function requiring shipment BOL.",
    category: "condition",
    code: bolExistsInlineConditionCode,
  },
  {
    id: "condition.pickup_stop.reference",
    label: "Pickup Stop Reference",
    description: "References include_pickup_stop from a script library.",
    category: "condition",
    code: "starlark:include_pickup_stop",
  },
  {
    id: "condition.pickup_stop.inline",
    label: "Pickup Stop Inline",
    description: "Defines an inline include function for LD or Pickup stops.",
    category: "condition",
    code: pickupStopInlineConditionCode,
  },
  {
    id: "condition.delivery_stop.reference",
    label: "Delivery Stop Reference",
    description: "References include_delivery_stop from a script library.",
    category: "condition",
    code: "starlark:include_delivery_stop",
  },
  {
    id: "condition.delivery_stop.inline",
    label: "Delivery Stop Inline",
    description: "Defines an inline include function for UL or Delivery stops.",
    category: "condition",
    code: deliveryStopInlineConditionCode,
  },
  {
    id: "library.condition_helpers",
    label: "Condition Helpers",
    description: "Adds reusable include functions for BOL, pickup stops, and delivery stops.",
    category: "scriptLibrary",
    code: conditionLibraryCode,
  },
  {
    id: "library.value_helpers",
    label: "Value Helpers",
    description: "Adds reusable shipment reference and contact phone helper functions.",
    category: "scriptLibrary",
    code: valueLibraryCode,
  },
];

export const ediScriptPresetsByCategory = ediScriptPresets.reduce(
  (groups, preset) => {
    groups[preset.category].push(preset);
    return groups;
  },
  {
    elementValue: [],
    repeatItem: [],
    condition: [],
    scriptLibrary: [],
  } as Record<EDIScriptPresetCategory, EDIScriptPreset[]>,
);

export function getEDIScriptPresetsByCategory(category: EDIScriptPresetCategory) {
  return ediScriptPresetsByCategory[category];
}

export function insertScriptPresetCode(current: string, preset: Pick<EDIScriptPreset, "code">) {
  const code = preset.code.trim();
  if (!current.trim()) return code;
  return `${current.trimEnd()}\n\n${code}`;
}
