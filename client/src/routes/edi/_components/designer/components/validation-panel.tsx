import { Button } from "@/components/ui/button";
import type { EDIDiagnostic } from "@/types/edi";
import { ListChecksIcon } from "lucide-react";
import { DiagnosticsList } from "./designer-shared";

export function ValidationPanel({
  diagnostics,
  onSelectDiagnostic,
  onValidate,
  isLoading,
  disabled,
}: {
  diagnostics: EDIDiagnostic[];
  onSelectDiagnostic: (diagnostic: EDIDiagnostic) => void;
  onValidate: () => void;
  isLoading: boolean;
  disabled: boolean;
}) {
  return (
    <div className="min-h-0 overflow-auto p-3">
      <div className="mb-3 flex items-center justify-between">
        <div>
          <div className="text-sm font-semibold">Validation Diagnostics</div>
          <div className="text-xs text-muted-foreground">
            {diagnostics.length} diagnostics returned by backend validation
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          onClick={onValidate}
          isLoading={isLoading}
          disabled={disabled}
        >
          <ListChecksIcon className="size-4" />
          Run
        </Button>
      </div>
      <DiagnosticsList diagnostics={diagnostics} onSelect={onSelectDiagnostic} />
    </div>
  );
}
