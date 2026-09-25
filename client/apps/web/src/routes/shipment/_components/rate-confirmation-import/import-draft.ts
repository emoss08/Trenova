import type {
  ImportDraftField,
  ImportDraftStop,
  PageDraft,
  PageDraftEdit,
} from "@/types/page-draft";
import { importRequiredFieldKeySchema } from "@/types/page-draft";
import { clampUnit, limitCodePoints } from "@trenova/shared/lib/utils";
import type { ReconciliationState, ReconciliationStop } from "./types";

/*
 * The bounds pagedraft.Draft.Validate enforces. A draft past any of them
 * refuses the whole turn, so the page sends what fits rather than nothing.
 */
const MAX_FIELDS = 80;
const MAX_STOPS = 20;
const MAX_FIELD_KEY_LENGTH = 64;
const MAX_FIELD_LABEL_LENGTH = 120;
const MAX_FIELD_VALUE_LENGTH = 1000;
const MAX_STOP_TEXT_LENGTH = 300;
const FIELD_KEY = /^[A-Za-z0-9_.-]+$/;

/** The cache key of the import assistant's conversation about a document. */
export function importAssistantConversationKey(documentId: string): readonly unknown[] {
  return ["import", documentId];
}

export type ImportRequiredValues = {
  customerId: string;
  serviceTypeId: string;
  shipmentTypeId: string;
  formulaTemplateId: string;
};

/** What the page does with a change the assistant hands it; the same as a person's click. */
export type ImportDraftHandlers = {
  acceptField: (key: string) => void;
  acceptAllConfident: () => void;
  editField: (key: string, value: string) => void;
  setRequiredField: (key: string, value: string) => void;
  setStopLocation: (stopIndex: number, locationId: string) => void;
  setStopSchedule: (stopIndex: number, windowStart: string, windowEnd?: string) => void;
};

function asText(value: unknown): string {
  if (value === null || value === undefined) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  try {
    return JSON.stringify(value) ?? "";
  } catch {
    return "";
  }
}

function stopText(value: unknown): string {
  return limitCodePoints(asText(value).trim(), MAX_STOP_TEXT_LENGTH);
}

function draftStop(stop: ReconciliationStop): ImportDraftStop {
  return {
    role: stop.role,
    name: stopText(stop.name.value),
    addressLine1: stopText(stop.addressLine1.value),
    city: stopText(stop.city.value),
    state: stopText(stop.state.value),
    postalCode: stopText(stop.postalCode.value),
    date: stopText(stop.date.value),
    timeWindow: stopText(stop.timeWindow.value),
    locationId: stop.locationId.trim(),
    confidence: clampUnit(stop.confidence),
  };
}

/** What the import page holds, in the shape and within the bounds the server reads. */
export function importDraftFromState(
  state: ReconciliationState,
  required: ImportRequiredValues,
): PageDraft {
  const fields: ImportDraftField[] = [];
  for (const [key, field] of Object.entries(state.fields)) {
    if (fields.length === MAX_FIELDS) {
      break;
    }
    if (key.length > MAX_FIELD_KEY_LENGTH || !FIELD_KEY.test(key)) {
      continue;
    }
    fields.push({
      key,
      label: limitCodePoints(field.label.trim(), MAX_FIELD_LABEL_LENGTH),
      value: limitCodePoints(asText(field.value).trim(), MAX_FIELD_VALUE_LENGTH),
      confidence: clampUnit(field.confidence),
      status: field.status,
    });
  }

  return {
    surface: "shipment_import",
    shipmentImport: {
      fields,
      required: {
        customerId: required.customerId.trim(),
        serviceTypeId: required.serviceTypeId.trim(),
        shipmentTypeId: required.shipmentTypeId.trim(),
        formulaTemplateId: required.formulaTemplateId.trim(),
      },
      stops: state.stops.slice(0, MAX_STOPS).map(draftStop),
    },
  };
}

function stopIndexOf(edit: PageDraftEdit, stopCount: number): number | null {
  const index = edit.stopIndex;
  if (index === null || index === undefined || index < 0 || index >= stopCount) {
    return null;
  }
  return index;
}

/**
 * Applies one change the import assistant handed the page. False when the
 * change no longer fits it — a stop that is gone, a field it does not name —
 * so nothing is half applied.
 */
export function applyImportDraftEdit(
  edit: PageDraftEdit,
  handlers: ImportDraftHandlers,
  stopCount: number,
): boolean {
  if (edit.surface !== "shipment_import") {
    return false;
  }
  const fieldKey = edit.fieldKey.trim();

  switch (edit.action) {
    case "accept_field":
      if (fieldKey === "") {
        return false;
      }
      handlers.acceptField(fieldKey);
      return true;
    case "accept_all_confident":
      handlers.acceptAllConfident();
      return true;
    case "set_field_value":
      if (fieldKey === "") {
        return false;
      }
      handlers.editField(fieldKey, edit.value);
      return true;
    case "set_required_field": {
      const key = importRequiredFieldKeySchema.safeParse(fieldKey);
      if (!key.success) {
        return false;
      }
      handlers.setRequiredField(key.data, edit.value);
      return true;
    }
    case "set_stop_location": {
      const index = stopIndexOf(edit, stopCount);
      if (index === null || edit.value.trim() === "") {
        return false;
      }
      handlers.setStopLocation(index, edit.value.trim());
      return true;
    }
    case "set_stop_schedule": {
      const index = stopIndexOf(edit, stopCount);
      if (index === null || edit.windowStart <= 0) {
        return false;
      }
      handlers.setStopSchedule(
        index,
        String(edit.windowStart),
        edit.windowEnd > 0 ? String(edit.windowEnd) : undefined,
      );
      return true;
    }
    default:
      return false;
  }
}

type CreateIssue = { path?: unknown; message?: unknown };

function isIssueList(value: unknown): value is CreateIssue[] {
  return (
    Array.isArray(value) &&
    value.length > 0 &&
    value.every(
      (issue) =>
        typeof issue === "object" &&
        issue !== null &&
        typeof (issue as CreateIssue).message === "string",
    )
  );
}

/**
 * Why creating the shipment failed, as lines the assistant can work through.
 * The create form reports schema issues as a JSON list of paths and messages;
 * anything else is passed on as it came.
 */
export function createFailureSummary(raw: string): string {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return raw;
  }
  if (!isIssueList(parsed)) {
    return raw;
  }

  return parsed
    .map((issue) => {
      const path = Array.isArray(issue.path) && issue.path.length > 0 ? issue.path.join(".") : "";
      return `- ${path === "" ? "shipment" : path}: ${String(issue.message)}`;
    })
    .join("\n");
}
