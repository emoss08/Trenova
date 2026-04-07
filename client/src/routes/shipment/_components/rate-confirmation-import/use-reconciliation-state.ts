import type { DocumentIntelligence } from "@/types/document";
import type { ShipmentCreateInput } from "@/types/shipment";
import { useCallback, useMemo, useReducer } from "react";
import {
  computeCounts,
  getEffectiveValue,
  reconciliationReducer,
  type ReconciliationCounts,
  type ReconciliationField,
  type ReconciliationState,
} from "./types";

const EMPTY_STATE: ReconciliationState = {
  fields: {},
  stops: [],
  overallConfidence: 0,
};

function parseInteger(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.trunc(value);
  }
  if (typeof value !== "string") return undefined;
  const digits = value.replace(/[^\d-]/g, "");
  if (!digits) return undefined;
  const parsed = Number.parseInt(digits, 10);
  return Number.isFinite(parsed) ? parsed : undefined;
}

function parseDecimal(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value !== "string") return 0;
  const normalized = value.replace(/[^0-9.-]/g, "");
  const parsed = Number.parseFloat(normalized);
  return Number.isFinite(parsed) ? parsed : 0;
}

function parseTimestamp(value?: string): number {
  if (!value?.trim()) return 0;

  // If the value is a pure number, it's already a Unix timestamp
  const trimmed = value.trim();
  if (/^\d+$/.test(trimmed)) {
    const num = Number(trimmed);
    if (num > 0) return num;
  }

  const parsed = Date.parse(trimmed);
  if (Number.isNaN(parsed)) return 0;
  return Math.floor(parsed / 1000);
}

function toStr(value: unknown): string {
  return typeof value === "string" ? value : value != null ? JSON.stringify(value) : "";
}

function addressLine(stop: {
  addressLine1: ReconciliationField;
  city: ReconciliationField;
  state: ReconciliationField;
  postalCode: ReconciliationField;
}): string {
  return [
    getEffectiveValue(stop.addressLine1),
    getEffectiveValue(stop.city),
    getEffectiveValue(stop.state),
    getEffectiveValue(stop.postalCode),
  ]
    .map((v) => (typeof v === "string" ? v.trim() : ""))
    .filter(Boolean)
    .join(", ");
}

export function useReconciliationState() {
  const [state, dispatch] = useReducer(reconciliationReducer, EMPTY_STATE);

  const initialize = useCallback(
    (data: DocumentIntelligence, missingRequired: string[]) => {
      dispatch({ type: "INIT", data, missingRequired });
    },
    [],
  );

  const acceptField = useCallback(
    (key: string) => dispatch({ type: "ACCEPT_FIELD", key }),
    [],
  );

  const editField = useCallback(
    (key: string, value: unknown) => dispatch({ type: "EDIT_FIELD", key, value }),
    [],
  );

  const resetField = useCallback(
    (key: string) => dispatch({ type: "RESET_FIELD", key }),
    [],
  );

  const acceptAllConfident = useCallback(
    () => dispatch({ type: "ACCEPT_ALL_CONFIDENT" }),
    [],
  );

  const acceptStopField = useCallback(
    (stopIndex: number, fieldKey: string) =>
      dispatch({ type: "ACCEPT_STOP_FIELD", stopIndex, fieldKey }),
    [],
  );

  const editStopField = useCallback(
    (stopIndex: number, fieldKey: string, value: unknown) =>
      dispatch({ type: "EDIT_STOP_FIELD", stopIndex, fieldKey, value }),
    [],
  );

  const setStopLocation = useCallback(
    (stopIndex: number, locationId: string) =>
      dispatch({ type: "SET_STOP_LOCATION", stopIndex, locationId }),
    [],
  );

  const counts: ReconciliationCounts = useMemo(() => computeCounts(state), [state]);

  const issueCount = counts.needsReview + counts.missing + counts.conflicting;

  const toShipmentCreateInput = useCallback(
    (requiredFields: {
      customerId: string;
      serviceTypeId: string;
      shipmentTypeId: string;
      formulaTemplateId: string;
      tractorTypeId?: string;
      trailerTypeId?: string;
    }): ShipmentCreateInput => {
      const fieldVal = (key: string) => {
        const f = state.fields[key];
        if (!f) return undefined;
        return getEffectiveValue(f);
      };

      const rate = parseDecimal(fieldVal("rate"));
      const weight = parseInteger(fieldVal("weight"));
      const pieces = parseInteger(fieldVal("pieces"));
      const bol = fieldVal("bol") ?? fieldVal("reference") ?? fieldVal("loadNumber");

      const moves =
        state.stops.length > 0
          ? [
              {
                status: "New" as const,
                loaded: true,
                sequence: 0,
                distance: 0,
                stops: state.stops.map((stop, index) => ({
                  status: "New" as const,
                  type: stop.role === "delivery" ? ("Delivery" as const) : ("Pickup" as const),
                  scheduleType: stop.appointmentRequired
                    ? ("Appointment" as const)
                    : ("Open" as const),
                  locationId: stop.locationId || "",
                  sequence: index,
                  scheduledWindowStart: parseTimestamp(
                    toStr(getEffectiveValue(stop.date)),
                  ),
                  scheduledWindowEnd: null,
                  addressLine: addressLine(stop),
                  weight: weight ?? null,
                  pieces: pieces ?? null,
                })),
              },
            ]
          : [
              {
                status: "New" as const,
                loaded: true,
                sequence: 0,
                distance: 0,
                stops: [
                  {
                    status: "New" as const,
                    type: "Pickup" as const,
                    scheduleType: "Open" as const,
                    locationId: "",
                    sequence: 0,
                    scheduledWindowStart: 0,
                    scheduledWindowEnd: null,
                    pieces: pieces ?? null,
                    weight: weight ?? null,
                  },
                  {
                    status: "New" as const,
                    type: "Delivery" as const,
                    scheduleType: "Open" as const,
                    locationId: "",
                    sequence: 1,
                    scheduledWindowStart: 0,
                    scheduledWindowEnd: null,
                    pieces: pieces ?? null,
                    weight: weight ?? null,
                  },
                ],
              },
            ];

      return {
        status: "New",
        bol: typeof bol === "string" ? bol : "",
        serviceTypeId: requiredFields.serviceTypeId,
        shipmentTypeId: requiredFields.shipmentTypeId,
        customerId: requiredFields.customerId,
        tractorTypeId: requiredFields.tractorTypeId ?? undefined,
        trailerTypeId: requiredFields.trailerTypeId ?? undefined,
        ownerId: undefined,
        enteredById: undefined,
        canceledById: undefined,
        formulaTemplateId: requiredFields.formulaTemplateId,
        consolidationGroupId: undefined,
        otherChargeAmount: 0,
        freightChargeAmount: rate,
        baseRate: rate,
        totalChargeAmount: rate,
        pieces: pieces ?? undefined,
        weight: weight ?? undefined,
        temperatureMin: undefined,
        temperatureMax: undefined,
        actualDeliveryDate: undefined,
        actualShipDate: undefined,
        canceledAt: undefined,
        ratingUnit: 1,
        additionalCharges: [],
        commodities: [],
        moves,
      };
    },
    [state],
  );

  return {
    state,
    counts,
    issueCount,
    initialize,
    acceptField,
    editField,
    resetField,
    acceptAllConfident,
    acceptStopField,
    editStopField,
    setStopLocation,
    toShipmentCreateInput,
    dispatch,
  };
}
