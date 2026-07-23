import type {
  EDIPartnerDocumentProfile,
  EDITemplate,
  PreviewEDIDocumentRequest,
} from "@/types/edi";

export type EDIDocumentSourceField =
  | "shipmentId"
  | "transferId"
  | "invoiceId"
  | "shipmentEventId"
  | "sourceMessageId"
  | "payload";

export type EDIDocumentSourceInput = {
  field: EDIDocumentSourceField;
  label: string;
  placeholder: string;
};

export type EDIDocumentSourceValues = Partial<Record<EDIDocumentSourceField, string>>;

export type EDIDocumentSourceContext = {
  transactionSet?: string;
  direction?: string;
};

export type EDIDocumentPayloadParseResult =
  | { ok: true; payload?: PreviewEDIDocumentRequest["payload"] }
  | { ok: false };

const ediDocumentSourceFields: EDIDocumentSourceField[] = [
  "shipmentId",
  "transferId",
  "invoiceId",
  "shipmentEventId",
  "sourceMessageId",
  "payload",
];

const ediDocumentSourceInputsByTransactionSet: Record<string, EDIDocumentSourceInput[]> = {
  "204": [
    { field: "shipmentId", label: "Shipment ID", placeholder: "sp_..." },
    { field: "transferId", label: "Transfer ID", placeholder: "editr_..." },
    { field: "payload", label: "Payload JSON", placeholder: "" },
  ],
  "210": [
    { field: "invoiceId", label: "Invoice ID", placeholder: "inv_..." },
    { field: "payload", label: "Payload JSON", placeholder: "" },
  ],
  "214": [
    { field: "shipmentId", label: "Shipment ID", placeholder: "sp_..." },
    { field: "shipmentEventId", label: "Shipment Event ID", placeholder: "se_..." },
    { field: "payload", label: "Payload JSON", placeholder: "" },
  ],
  "990": [
    { field: "transferId", label: "Transfer ID", placeholder: "editr_..." },
    { field: "payload", label: "Payload JSON", placeholder: "" },
  ],
  "997": [
    { field: "sourceMessageId", label: "Source Message ID", placeholder: "edimsg_..." },
    { field: "payload", label: "Payload JSON", placeholder: "" },
  ],
  "999": [
    { field: "sourceMessageId", label: "Source Message ID", placeholder: "edimsg_..." },
    { field: "payload", label: "Payload JSON", placeholder: "" },
  ],
};

export function getEDIDocumentSourceInputs(transactionSet?: string | null) {
  return (
    ediDocumentSourceInputsByTransactionSet[transactionSet ?? ""] ?? [
      { field: "payload", label: "Payload JSON", placeholder: "" },
    ]
  );
}

export function hasEDIDocumentSourceValue(
  values: EDIDocumentSourceValues,
  transactionSet?: string | null,
) {
  return getEDIDocumentSourceInputs(transactionSet).some((input) =>
    Boolean(values[input.field]?.trim()),
  );
}

export function pruneEDIDocumentSourceValues(
  values: EDIDocumentSourceValues,
  transactionSet?: string | null,
) {
  const activeFields = new Set(
    getEDIDocumentSourceInputs(transactionSet).map((input) => input.field),
  );
  let changed = false;
  const nextValues: EDIDocumentSourceValues = {};

  for (const field of ediDocumentSourceFields) {
    const value = values[field];
    if (value === undefined) continue;
    if (activeFields.has(field)) {
      nextValues[field] = value;
      continue;
    }
    changed = true;
  }

  return changed ? nextValues : values;
}

export function buildEDIDocumentSourceRequest(
  values: EDIDocumentSourceValues,
  transactionSet?: string | null,
  direction?: string | null,
): Pick<
  PreviewEDIDocumentRequest,
  | "shipmentId"
  | "transferId"
  | "invoiceId"
  | "shipmentEventId"
  | "sourceMessageId"
  | "transactionSet"
  | "direction"
> {
  const activeFields = new Set(
    getEDIDocumentSourceInputs(transactionSet).map((input) => input.field),
  );
  return {
    shipmentId: activeFields.has("shipmentId") ? trimmedSourceValue(values.shipmentId) : undefined,
    transferId: activeFields.has("transferId") ? trimmedSourceValue(values.transferId) : undefined,
    invoiceId: activeFields.has("invoiceId") ? trimmedSourceValue(values.invoiceId) : undefined,
    shipmentEventId: activeFields.has("shipmentEventId")
      ? trimmedSourceValue(values.shipmentEventId)
      : undefined,
    sourceMessageId: activeFields.has("sourceMessageId")
      ? trimmedSourceValue(values.sourceMessageId)
      : undefined,
    transactionSet: ediTransactionSetValue(transactionSet),
    direction: ediDirectionValue(direction),
  };
}

export function buildEDIDocumentResolutionRequest({
  partnerDocumentProfileId,
  ediPartnerId,
  sourceValues,
  transactionSet,
  direction,
  payload,
}: {
  partnerDocumentProfileId?: string | null;
  ediPartnerId?: string | null;
  sourceValues: EDIDocumentSourceValues;
  transactionSet?: string | null;
  direction?: string | null;
  payload?: PreviewEDIDocumentRequest["payload"];
}): PreviewEDIDocumentRequest {
  return {
    partnerDocumentProfileId: trimmedSourceValue(partnerDocumentProfileId ?? undefined),
    ediPartnerId: trimmedSourceValue(ediPartnerId ?? undefined),
    ...buildEDIDocumentSourceRequest(sourceValues, transactionSet, direction),
    payload,
  };
}

export function resolveEDIDocumentSourceContext({
  profile,
  template,
  fallbackTransactionSet,
  fallbackDirection,
}: {
  profile?: EDIPartnerDocumentProfile | null;
  template?: EDITemplate | null;
  fallbackTransactionSet?: string | null;
  fallbackDirection?: string | null;
}): EDIDocumentSourceContext {
  return {
    transactionSet:
      template?.transactionSet || profile?.transactionSet || fallbackTransactionSet || undefined,
    direction: template?.direction || profile?.direction || fallbackDirection || undefined,
  };
}

export function parseEDIDocumentPayload(value: string): EDIDocumentPayloadParseResult {
  if (!value.trim()) return { ok: true };
  try {
    return { ok: true, payload: JSON.parse(value) as PreviewEDIDocumentRequest["payload"] };
  } catch {
    return { ok: false };
  }
}

function trimmedSourceValue(value?: string) {
  const trimmed = value?.trim();
  return trimmed ? trimmed : undefined;
}

function ediTransactionSetValue(
  value?: string | null,
): PreviewEDIDocumentRequest["transactionSet"] {
  switch (value) {
    case "204":
    case "210":
    case "214":
    case "990":
    case "997":
    case "999":
      return value;
    default:
      return undefined;
  }
}

function ediDirectionValue(value?: string | null): PreviewEDIDocumentRequest["direction"] {
  switch (value) {
    case "Inbound":
    case "Outbound":
      return value;
    default:
      return undefined;
  }
}
