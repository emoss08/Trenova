import type {
  DocumentIntelligence,
  DocumentIntelligenceConflict,
  DocumentIntelligenceField,
  DocumentIntelligenceStop,
} from "@/types/document";

export type RequiredFieldsForm = {
  customerId: string;
  serviceTypeId: string;
  shipmentTypeId: string;
  formulaTemplateId: string;
  tractorTypeId: string;
  trailerTypeId: string;
  stops: Array<{ locationId: string }>;
};

export type FieldStatus = "accepted" | "needs-review" | "missing" | "conflicting" | "edited";

export type ReconciliationField = {
  key: string;
  label: string;
  value: unknown;
  confidence: number;
  status: FieldStatus;
  evidenceExcerpt?: string;
  pageNumber?: number;
  alternativeValues?: string[];
  conflict?: { values: string[]; pageNumbers: number[] };
  editedValue?: unknown;
  originalValue?: unknown;
};

export type ReconciliationStop = {
  sequence: number;
  role: "pickup" | "delivery";
  status: FieldStatus;
  confidence: number;
  name: ReconciliationField;
  addressLine1: ReconciliationField;
  city: ReconciliationField;
  state: ReconciliationField;
  postalCode: ReconciliationField;
  date: ReconciliationField;
  timeWindow: ReconciliationField;
  locationId: string;
  appointmentRequired: boolean;
  evidenceExcerpt?: string;
  pageNumber?: number;
};

export type ReconciliationCounts = {
  accepted: number;
  needsReview: number;
  missing: number;
  conflicting: number;
  edited: number;
  total: number;
};

export type ReconciliationPhase = "upload" | "processing" | "reconciliation" | "success";

export type ReconciliationAction =
  | { type: "ACCEPT_FIELD"; key: string }
  | { type: "EDIT_FIELD"; key: string; value: unknown }
  | { type: "RESET_FIELD"; key: string }
  | { type: "ACCEPT_ALL_CONFIDENT" }
  | { type: "ACCEPT_STOP_FIELD"; stopIndex: number; fieldKey: string }
  | { type: "EDIT_STOP_FIELD"; stopIndex: number; fieldKey: string; value: unknown }
  | { type: "SET_STOP_LOCATION"; stopIndex: number; locationId: string }
  | { type: "INIT"; data: DocumentIntelligence; missingRequired: string[] };

export type ReconciliationState = {
  fields: Record<string, ReconciliationField>;
  stops: ReconciliationStop[];
  overallConfidence: number;
};

const HIGH_CONFIDENCE_THRESHOLD = 0.85;
const LOW_CONFIDENCE_THRESHOLD = 0.5;

const FIELD_LABELS: Record<string, string> = {
  loadNumber: "Load Number",
  referenceNumber: "Reference Number",
  shipper: "Shipper",
  consignee: "Consignee",
  rate: "Rate",
  equipmentType: "Equipment Type",
  commodity: "Commodity",
  pickupDate: "Pickup Date",
  deliveryDate: "Delivery Date",
  pickupWindow: "Pickup Window",
  deliveryWindow: "Delivery Window",
  pickupNumber: "Pickup Number",
  deliveryNumber: "Delivery Number",
  appointmentNumber: "Appointment Number",
  bol: "BOL",
  poNumber: "PO Number",
  scac: "SCAC",
  proNumber: "Pro Number",
  paymentTerms: "Payment Terms",
  billTo: "Bill To",
  carrierName: "Carrier Name",
  carrierContact: "Carrier Contact",
  containerNumber: "Container Number",
  trailerNumber: "Trailer Number",
  tractorNumber: "Tractor Number",
  fuelSurcharge: "Fuel Surcharge",
  serviceType: "Service Type",
  weight: "Weight",
  pieces: "Pieces",
};

function computeFieldStatus(
  field: DocumentIntelligenceField,
  conflict: DocumentIntelligenceConflict | undefined,
  isMissingRequired: boolean,
): FieldStatus {
  if (isMissingRequired || field.value == null || field.value === "") {
    return "missing";
  }
  if (conflict || field.conflict) {
    return "conflicting";
  }
  if (field.reviewRequired) {
    return "needs-review";
  }
  const confidence = field.confidence ?? 0;
  if (confidence >= HIGH_CONFIDENCE_THRESHOLD) {
    return "accepted";
  }
  if (confidence >= LOW_CONFIDENCE_THRESHOLD) {
    return "needs-review";
  }
  return "missing";
}

function fieldFromIntelligence(
  key: string,
  field: DocumentIntelligenceField,
  conflict: DocumentIntelligenceConflict | undefined,
  isMissingRequired: boolean,
): ReconciliationField {
  return {
    key,
    label: field.label || FIELD_LABELS[key] || key,
    value: field.value,
    confidence: field.confidence ?? 0,
    status: computeFieldStatus(field, conflict, isMissingRequired),
    evidenceExcerpt: field.evidenceExcerpt || field.excerpt,
    pageNumber: field.pageNumber,
    alternativeValues: conflict?.values,
    conflict: conflict
      ? { values: conflict.values ?? [], pageNumbers: conflict.pageNumbers ?? [] }
      : undefined,
    originalValue: field.value,
  };
}

function computeStopStatus(stop: DocumentIntelligenceStop): FieldStatus {
  if (stop.reviewRequired) return "needs-review";
  const confidence = stop.confidence ?? 0;
  if (confidence >= HIGH_CONFIDENCE_THRESHOLD) return "accepted";
  if (confidence >= LOW_CONFIDENCE_THRESHOLD) return "needs-review";
  return "missing";
}

function stopFieldFromValue(key: string, value: string | undefined, confidence: number): ReconciliationField {
  const hasValue = !!value && value.trim() !== "";
  return {
    key,
    label: FIELD_LABELS[key] || key,
    value: value ?? "",
    confidence: hasValue ? confidence : 0,
    status: hasValue ? (confidence >= HIGH_CONFIDENCE_THRESHOLD ? "accepted" : "needs-review") : "missing",
    originalValue: value ?? "",
  };
}

function stopFromIntelligence(stop: DocumentIntelligenceStop): ReconciliationStop {
  const confidence = stop.confidence ?? 0;
  return {
    sequence: stop.sequence,
    role: stop.role === "delivery" ? "delivery" : "pickup",
    status: computeStopStatus(stop),
    confidence,
    name: stopFieldFromValue("name", stop.name, confidence),
    addressLine1: stopFieldFromValue("addressLine1", stop.addressLine1, confidence),
    city: stopFieldFromValue("city", stop.city, confidence),
    state: stopFieldFromValue("state", stop.state, confidence),
    postalCode: stopFieldFromValue("postalCode", stop.postalCode, confidence),
    date: stopFieldFromValue("date", stop.date, confidence),
    timeWindow: stopFieldFromValue("timeWindow", stop.timeWindow, confidence),
    locationId: "",
    appointmentRequired: stop.appointmentRequired ?? false,
    evidenceExcerpt: stop.evidenceExcerpt,
    pageNumber: stop.pageNumber,
  };
}

export function initializeReconciliation(
  data: DocumentIntelligence,
  missingRequired: string[],
): ReconciliationState {
  const conflictMap = new Map<string, DocumentIntelligenceConflict>();
  for (const c of data.conflicts ?? []) {
    if (c.key) conflictMap.set(c.key, c);
  }

  const missingSet = new Set(missingRequired);

  const fields: Record<string, ReconciliationField> = {};
  for (const [key, field] of Object.entries(data.fields ?? {})) {
    fields[key] = fieldFromIntelligence(key, field, conflictMap.get(key), missingSet.has(key));
  }

  for (const missing of missingRequired) {
    if (!fields[missing]) {
      fields[missing] = {
        key: missing,
        label: FIELD_LABELS[missing] || missing,
        value: undefined,
        confidence: 0,
        status: "missing",
        originalValue: undefined,
      };
    }
  }

  const stops = (data.stops ?? []).map(stopFromIntelligence);

  return {
    fields,
    stops,
    overallConfidence: data.overallConfidence ?? 0,
  };
}

export function reconciliationReducer(
  state: ReconciliationState,
  action: ReconciliationAction,
): ReconciliationState {
  switch (action.type) {
    case "INIT": {
      return initializeReconciliation(action.data, action.missingRequired);
    }

    case "ACCEPT_FIELD": {
      const field = state.fields[action.key];
      if (!field) return state;
      return {
        ...state,
        fields: {
          ...state.fields,
          [action.key]: { ...field, status: "accepted" },
        },
      };
    }

    case "EDIT_FIELD": {
      const field = state.fields[action.key];
      if (!field) return state;
      return {
        ...state,
        fields: {
          ...state.fields,
          [action.key]: {
            ...field,
            editedValue: action.value,
            status: "edited",
          },
        },
      };
    }

    case "RESET_FIELD": {
      const field = state.fields[action.key];
      if (!field) return state;
      const restoredValue = field.originalValue;
      const hasValue = restoredValue != null && restoredValue !== "";
      return {
        ...state,
        fields: {
          ...state.fields,
          [action.key]: {
            ...field,
            editedValue: undefined,
            status: hasValue
              ? field.confidence >= HIGH_CONFIDENCE_THRESHOLD
                ? "accepted"
                : "needs-review"
              : "missing",
          },
        },
      };
    }

    case "ACCEPT_ALL_CONFIDENT": {
      const newFields = { ...state.fields };
      for (const [key, field] of Object.entries(newFields)) {
        if (
          field.status === "needs-review" &&
          field.confidence >= HIGH_CONFIDENCE_THRESHOLD &&
          !field.conflict
        ) {
          newFields[key] = { ...field, status: "accepted" };
        }
      }
      const newStops = state.stops.map((stop) => {
        if (stop.confidence >= HIGH_CONFIDENCE_THRESHOLD && stop.status === "needs-review") {
          return { ...stop, status: "accepted" as FieldStatus };
        }
        return stop;
      });
      return { ...state, fields: newFields, stops: newStops };
    }

    case "ACCEPT_STOP_FIELD": {
      const stop = state.stops[action.stopIndex];
      if (!stop) return state;
      const fieldKey = action.fieldKey as keyof ReconciliationStop;
      const field = stop[fieldKey];
      if (!field || typeof field !== "object" || !("status" in field)) return state;
      const newStops = [...state.stops];
      newStops[action.stopIndex] = {
        ...stop,
        [action.fieldKey]: { ...field, status: "accepted" },
      };
      return { ...state, stops: newStops };
    }

    case "EDIT_STOP_FIELD": {
      const stop = state.stops[action.stopIndex];
      if (!stop) return state;
      const fieldKey = action.fieldKey as keyof ReconciliationStop;
      const field = stop[fieldKey];
      if (!field || typeof field !== "object" || !("status" in field)) return state;
      const newStops = [...state.stops];
      newStops[action.stopIndex] = {
        ...stop,
        [action.fieldKey]: { ...field, editedValue: action.value, status: "edited" },
      };
      return { ...state, stops: newStops };
    }

    case "SET_STOP_LOCATION": {
      const stop = state.stops[action.stopIndex];
      if (!stop) return state;
      const newStops = [...state.stops];
      newStops[action.stopIndex] = { ...stop, locationId: action.locationId };
      return { ...state, stops: newStops };
    }

    default:
      return state;
  }
}

export function computeCounts(state: ReconciliationState): ReconciliationCounts {
  const counts: ReconciliationCounts = {
    accepted: 0,
    needsReview: 0,
    missing: 0,
    conflicting: 0,
    edited: 0,
    total: 0,
  };

  for (const field of Object.values(state.fields)) {
    counts.total++;
    switch (field.status) {
      case "accepted":
        counts.accepted++;
        break;
      case "needs-review":
        counts.needsReview++;
        break;
      case "missing":
        counts.missing++;
        break;
      case "conflicting":
        counts.conflicting++;
        break;
      case "edited":
        counts.edited++;
        break;
    }
  }

  return counts;
}

export function getEffectiveValue(field: ReconciliationField): unknown {
  return field.editedValue !== undefined ? field.editedValue : field.value;
}
