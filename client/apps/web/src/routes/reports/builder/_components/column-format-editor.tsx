import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Switch } from "@trenova/shared/components/ui/switch";
import { formatReportValue } from "@trenova/shared/lib/report-format";
import {
  MAX_TRANSFORM_PRECISION,
  bandIsSet,
  REPORT_BOOL_STYLE_CHOICES,
  REPORT_CURRENCY_CHOICES,
  REPORT_DATE_STYLE_CHOICES,
  REPORT_DURATION_STYLE_CHOICES,
  REPORT_DURATION_UNIT_CHOICES,
  REPORT_NEGATIVE_CHOICES,
  REPORT_NOTATION_CHOICES,
  columnValueType,
  displayStylesForType,
  transformChoicesForType,
  transformUsesPrecision,
  type ReportColumnSpec,
  type ReportDisplaySpec,
  type ReportTransformOp,
} from "@/types/report";
import { RotateCcwIcon } from "lucide-react";
import { DisplayRulesEditor } from "./display-rules-editor";
import { resolvePreviewDisplay, sampleValueFor } from "./display-preview";

type ColumnFormatEditorProps = {
  column: ReportColumnSpec;
  fieldType: string | undefined;
  formatHint: string | undefined;
  onUpdate: (column: ReportColumnSpec) => void;
};

const NONE = "__none__";

function SelectField<T extends string>({
  label,
  value,
  choices,
  placeholder,
  onChange,
}: {
  label: string;
  value: string;
  choices: { value: T; label: string }[];
  placeholder?: string;
  onChange: (value: T | undefined) => void;
}) {
  const items = [{ value: NONE, label: placeholder ?? "Default" }, ...choices];

  return (
    <div className="flex flex-col gap-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <Select
        value={value || NONE}
        onValueChange={(next) => {
          if (!next) return;
          onChange(next === NONE ? undefined : (next as T));
        }}
        items={items}
      >
        <SelectTrigger className="h-7">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {items.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

export function ColumnFormatEditor({
  column,
  fieldType,
  formatHint,
  onUpdate,
}: ColumnFormatEditorProps) {
  const valueType = columnValueType(column, fieldType);
  const display = column.display ?? {};
  const transform = column.transform;
  const transformChoices = transformChoicesForType(valueType);

  const resolved = resolvePreviewDisplay(valueType, formatHint, column, display);
  const preview = formatReportValue(sampleValueFor(resolved, valueType), resolved);

  const patchDisplay = (patch: Partial<ReportDisplaySpec>) => {
    const next = { ...display, ...patch };
    const cleaned = Object.fromEntries(
      Object.entries(next).filter(([, value]) => value !== undefined && value !== ""),
    ) as ReportDisplaySpec;
    onUpdate({
      ...column,
      display: Object.keys(cleaned).length > 0 ? cleaned : undefined,
    });
  };

  const setTransform = (op: ReportTransformOp | undefined) => {
    if (!op) {
      onUpdate({ ...column, transform: undefined });
      return;
    }
    onUpdate({
      ...column,
      transform: {
        op,
        precision: transformUsesPrecision(op) ? (transform?.precision ?? 2) : undefined,
        factor: op === "scale" ? (transform?.factor ?? 1) : undefined,
      },
    });
  };

  const numeric =
    resolved.style === "number" || resolved.style === "currency" || resolved.style === "percent";

  return (
    <div className="flex flex-col gap-2 rounded-md border border-dashed border-border bg-muted/30 p-2">
      <div className="flex items-center gap-2">
        <span className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
          Formatting
        </span>
        <span className="rounded-sm bg-background px-1.5 py-px font-mono text-2xs text-foreground/80 tabular-nums">
          {preview || "—"}
        </span>
        <div className="flex-1" />
        {(column.display || column.transform) && (
          <Button
            variant="ghost"
            size="icon"
            className="size-6"
            aria-label="Reset formatting"
            onClick={() => onUpdate({ ...column, display: undefined, transform: undefined })}
          >
            <RotateCcwIcon className="size-3.5" />
          </Button>
        )}
      </div>

      <div className="grid grid-cols-2 gap-2">
        <SelectField
          label="Display as"
          value={display.style ?? ""}
          choices={displayStylesForType(valueType)}
          onChange={(style) => patchDisplay({ style })}
        />

        {numeric && (
          <div className="flex flex-col gap-1">
            <Label className="text-xs text-muted-foreground">Decimal places</Label>
            <Input
              className="h-7"
              type="number"
              min={0}
              max={10}
              value={display.decimals ?? ""}
              placeholder={String(resolved.decimals)}
              onChange={(event) =>
                patchDisplay({
                  decimals: event.target.value === "" ? undefined : Number(event.target.value),
                })
              }
            />
          </div>
        )}

        {resolved.style === "currency" && (
          <SelectField
            label="Currency"
            value={display.currency ?? ""}
            choices={REPORT_CURRENCY_CHOICES}
            onChange={(currency) => patchDisplay({ currency })}
          />
        )}

        {numeric && (
          <SelectField
            label="Negatives"
            value={display.negative ?? ""}
            choices={REPORT_NEGATIVE_CHOICES}
            onChange={(negative) => patchDisplay({ negative })}
          />
        )}

        {numeric && (
          <SelectField
            label="Large numbers"
            value={display.notation ?? ""}
            choices={REPORT_NOTATION_CHOICES}
            onChange={(notation) => patchDisplay({ notation })}
          />
        )}

        {resolved.style === "date" && (
          <SelectField
            label="Date format"
            value={display.dateStyle ?? ""}
            choices={REPORT_DATE_STYLE_CHOICES}
            onChange={(dateStyle) => patchDisplay({ dateStyle })}
          />
        )}

        {resolved.style === "bool" && (
          <SelectField
            label="Values shown as"
            value={display.boolStyle ?? ""}
            choices={REPORT_BOOL_STYLE_CHOICES}
            onChange={(boolStyle) => patchDisplay({ boolStyle })}
          />
        )}

        {resolved.style === "duration" && (
          <SelectField
            label="Source unit"
            value={display.durationUnit ?? ""}
            choices={REPORT_DURATION_UNIT_CHOICES}
            onChange={(durationUnit) => patchDisplay({ durationUnit })}
          />
        )}

        {resolved.style === "duration" && (
          <SelectField
            label="Duration format"
            value={display.durationStyle ?? ""}
            choices={REPORT_DURATION_STYLE_CHOICES}
            onChange={(durationStyle) => patchDisplay({ durationStyle })}
          />
        )}

        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Prefix</Label>
          <Input
            className="h-7"
            maxLength={12}
            value={display.prefix ?? ""}
            placeholder={resolved.prefix || "None"}
            onChange={(event) => patchDisplay({ prefix: event.target.value || undefined })}
          />
        </div>

        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Suffix</Label>
          <Input
            className="h-7"
            maxLength={12}
            value={display.suffix ?? ""}
            placeholder={resolved.suffix || "None"}
            onChange={(event) => patchDisplay({ suffix: event.target.value || undefined })}
          />
        </div>

        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Empty values show</Label>
          <Input
            className="h-7"
            maxLength={24}
            value={display.nullText ?? ""}
            placeholder="Blank"
            onChange={(event) => patchDisplay({ nullText: event.target.value || undefined })}
          />
        </div>

        {numeric && (
          <div className="flex items-center justify-between gap-2 self-end rounded-md border border-border bg-background px-2 py-1">
            <Label className="text-xs text-muted-foreground">Thousands separator</Label>
            <Switch
              checked={display.grouping ?? resolved.grouping}
              onCheckedChange={(checked) => patchDisplay({ grouping: checked })}
            />
          </div>
        )}
      </div>

      {numeric && (
        <DisplayRulesEditor
          rules={display.rules ?? []}
          onChange={(rules) => patchDisplay({ rules })}
        />
      )}

      {transformChoices.length > 0 && !bandIsSet(column.band) && (
        <div className="grid grid-cols-2 gap-2 border-t border-border/60 pt-2">
          <SelectField
            label="Transform value"
            value={transform?.op ?? ""}
            choices={transformChoices}
            placeholder="No transform"
            onChange={(op) => setTransform(op)}
          />

          {transform && transformUsesPrecision(transform.op) && (
            <div className="flex flex-col gap-1">
              <Label className="text-xs text-muted-foreground">Decimal places</Label>
              <Input
                className="h-7"
                type="number"
                min={0}
                max={MAX_TRANSFORM_PRECISION}
                value={transform.precision ?? 0}
                onChange={(event) =>
                  onUpdate({
                    ...column,
                    transform: { ...transform, precision: Number(event.target.value) },
                  })
                }
              />
            </div>
          )}

          {transform?.op === "scale" && (
            <div className="flex flex-col gap-1">
              <Label className="text-xs text-muted-foreground">Multiplier</Label>
              <Input
                className="h-7"
                type="number"
                step="any"
                value={transform.factor ?? 1}
                onChange={(event) =>
                  onUpdate({
                    ...column,
                    transform: { ...transform, factor: Number(event.target.value) },
                  })
                }
              />
            </div>
          )}

          <p className="col-span-2 text-2xs text-muted-foreground">
            Transforms change the number this report returns — sorting and exported cells use the
            transformed value. Your records are never modified.
          </p>
        </div>
      )}
    </div>
  );
}
