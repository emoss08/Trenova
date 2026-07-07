import {
  ControlledEDITransferAutocompleteField,
  ControlledShipmentAutocompleteField,
} from "@/components/autocomplete-fields";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  getEDIDocumentSourceInputs,
  type EDIDocumentSourceField,
  type EDIDocumentSourceValues,
} from "@/lib/edi/document-source";

type DocumentSourceControlsLayout = "stack" | "toolbar";

type DocumentSourceControlsProps = {
  transactionSet?: string | null;
  values: EDIDocumentSourceValues;
  onChange: (field: EDIDocumentSourceField, value: string) => void;
  layout?: DocumentSourceControlsLayout;
};

export function DocumentSourceControls({
  transactionSet,
  values,
  onChange,
  layout = "stack",
}: DocumentSourceControlsProps) {
  const sourceInputs = getEDIDocumentSourceInputs(transactionSet);

  return (
    <div className="flex flex-row gap-1">
      {sourceInputs.map((input) => {
        const value = values[input.field] ?? "";
        if (input.field === "payload") {
          return layout === "toolbar" ? (
            <DocumentSourceField
              key={input.field}
              label={input.label}
              value={value}
              placeholder='{"transactionSet":"204"}'
              onChange={(nextValue) => onChange(input.field, nextValue)}
              className="w-64"
            />
          ) : (
            <DocumentSourceTextarea
              key={input.field}
              label={input.label}
              value={value}
              onChange={(nextValue) => onChange(input.field, nextValue)}
            />
          );
        }

        const fieldClassName = layout === "toolbar" ? "w-56" : undefined;

        if (input.field === "shipmentId") {
          return (
            <div key={input.field} className={fieldClassName}>
              <ControlledShipmentAutocompleteField
                label={input.label}
                value={value}
                onValueChange={(nextValue) => onChange(input.field, nextValue)}
              />
            </div>
          );
        }

        if (input.field === "transferId") {
          return (
            <div key={input.field} className={fieldClassName}>
              <ControlledEDITransferAutocompleteField
                label={input.label}
                value={value}
                onValueChange={(nextValue) => onChange(input.field, nextValue)}
              />
            </div>
          );
        }

        return (
          <DocumentSourceField
            key={input.field}
            label={input.label}
            placeholder={input.placeholder}
            value={value}
            onChange={(nextValue) => onChange(input.field, nextValue)}
            className={fieldClassName}
          />
        );
      })}
    </div>
  );
}

function DocumentSourceField({
  label,
  value,
  onChange,
  placeholder,
  className,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}) {
  return (
    <div className={className}>
      <div className="space-y-1">
        <Label className="text-xs text-muted-foreground">{label}</Label>
        <Input
          value={value}
          placeholder={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />
      </div>
    </div>
  );
}

function DocumentSourceTextarea({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <Textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="min-h-24 font-mono text-xs"
      />
    </div>
  );
}
