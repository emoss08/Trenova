import { LocationAutocompleteField } from "@/components/autocomplete-fields";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { MapPinIcon, TruckIcon } from "lucide-react";
import { useCallback, useState } from "react";
import type { Control, Path } from "react-hook-form";
import {
  type FieldStatus,
  type ReconciliationField,
  type ReconciliationStop,
  type RequiredFieldsForm,
  getEffectiveValue,
} from "./types";

type StopReconciliationCardProps = {
  stop: ReconciliationStop;
  index: number;
  onEditField: (stopIndex: number, fieldKey: string, value: unknown) => void;
  formControl: Control<RequiredFieldsForm>;
  locationFieldName: Path<RequiredFieldsForm>;
};

const DOT_STYLE: Record<FieldStatus, string> = {
  accepted: "bg-emerald-500",
  "needs-review": "bg-amber-500",
  missing: "bg-muted-foreground/20",
  conflicting: "bg-amber-500",
  edited: "bg-blue-500",
};

function InlineField({
  label,
  value,
  status,
  onEdit,
}: {
  label: string;
  value: string;
  status: FieldStatus;
  onEdit: (value: string) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [editVal, setEditVal] = useState(value);

  const save = useCallback(() => {
    setEditing(false);
    if (editVal !== value) onEdit(editVal);
  }, [editVal, value, onEdit]);

  return (
    <div className="flex items-center gap-2 py-px">
      <div className={cn("size-1 shrink-0 rounded-full", DOT_STYLE[status])} />
      <span className="w-12 shrink-0 text-2xs text-muted-foreground">{label}</span>
      {editing ? (
        <Input
          value={editVal}
          onChange={(e) => setEditVal(e.target.value)}
          onBlur={save}
          onKeyDown={(e) => {
            if (e.key === "Enter") save();
            if (e.key === "Escape") setEditing(false);
          }}
          className="h-5 flex-1 text-xs"
          autoFocus
        />
      ) : (
        <span
          className="flex-1 cursor-text truncate text-xs"
          onClick={() => {
            setEditVal(value);
            setEditing(true);
          }}
        >
          {value}
        </span>
      )}
    </div>
  );
}

function toStr(field: ReconciliationField) {
  const v = getEffectiveValue(field);
  return typeof v === "string" ? v : v != null ? JSON.stringify(v) : "";
}

export function StopReconciliationCard({
  stop,
  index,
  onEditField,
  formControl,
  locationFieldName,
}: StopReconciliationCardProps) {
  const isPickup = stop.role === "pickup";

  // Build address string for display
  const nameVal = toStr(stop.name);
  const addr = [toStr(stop.addressLine1), toStr(stop.city), toStr(stop.state), toStr(stop.postalCode)]
    .filter(Boolean)
    .join(", ");
  const dateVal = toStr(stop.date);
  const timeVal = toStr(stop.timeWindow);

  // Collect non-empty fields for inline editing
  const editableFields: Array<{ key: string; label: string; value: string; field: ReconciliationField }> = [];
  if (nameVal) editableFields.push({ key: "name", label: "Name", value: nameVal, field: stop.name });
  if (toStr(stop.addressLine1)) editableFields.push({ key: "addressLine1", label: "Address", value: toStr(stop.addressLine1), field: stop.addressLine1 });
  if (toStr(stop.city)) editableFields.push({ key: "city", label: "City", value: toStr(stop.city), field: stop.city });
  if (toStr(stop.state)) editableFields.push({ key: "state", label: "State", value: toStr(stop.state), field: stop.state });
  if (toStr(stop.postalCode)) editableFields.push({ key: "postalCode", label: "Zip", value: toStr(stop.postalCode), field: stop.postalCode });
  if (dateVal) editableFields.push({ key: "date", label: "Date", value: dateVal, field: stop.date });
  if (timeVal) editableFields.push({ key: "timeWindow", label: "Window", value: timeVal, field: stop.timeWindow });

  return (
    <div className="rounded-lg border p-3">
      {/* Header */}
      <div className="mb-2 flex items-center justify-between">
        <div className="flex items-center gap-2">
          {isPickup ? (
            <TruckIcon className="size-3.5 text-muted-foreground" />
          ) : (
            <MapPinIcon className="size-3.5 text-muted-foreground" />
          )}
          <span className="text-xs font-medium">
            {isPickup ? "Pickup" : "Delivery"} {stop.sequence + 1}
          </span>
          {stop.confidence > 0 && (
            <span className="text-2xs text-muted-foreground/50 tabular-nums">
              {Math.round(stop.confidence * 100)}%
            </span>
          )}
        </div>
        <div className="flex items-center gap-1.5">
          {stop.appointmentRequired && (
            <Badge variant="outline" className="h-4 px-1 text-2xs">Appt</Badge>
          )}
        </div>
      </div>

      {/* Primary info: name + address */}
      {(nameVal || addr) && (
        <div className="mb-2">
          {nameVal && <div className="text-xs font-medium">{nameVal}</div>}
          {addr && <div className="text-2xs text-muted-foreground">{addr}</div>}
          {(dateVal || timeVal) && (
            <div className="mt-0.5 text-2xs text-muted-foreground/60">
              {[dateVal, timeVal].filter(Boolean).join(" · ")}
            </div>
          )}
        </div>
      )}

      {/* Editable fields (only non-empty) */}
      {editableFields.length > 0 && (
        <details className="group">
          <summary className="cursor-pointer text-2xs text-muted-foreground/40 transition-colors hover:text-muted-foreground">
            Edit fields
          </summary>
          <div className="mt-1.5 space-y-px">
            {editableFields.map((f) => (
              <InlineField
                key={f.key}
                label={f.label}
                value={f.value}
                status={f.field.status}
                onEdit={(v) => onEditField(index, f.key, v)}
              />
            ))}
          </div>
        </details>
      )}

      {/* Location match */}
      <div className="mt-2 border-t pt-2">
        <LocationAutocompleteField
          control={formControl}
          name={locationFieldName}
          placeholder="Match to location..."
          clearable
        />
      </div>
    </div>
  );
}
