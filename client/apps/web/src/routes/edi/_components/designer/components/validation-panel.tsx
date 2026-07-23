import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { ListChecksIcon } from "lucide-react";
import {
  useSelectDiagnostic,
  useTemplateDesignerValidationAction,
} from "@/hooks/use-template-designer-state";
import { useTemplateDesignerStore } from "@/stores/template-designer-store";
import { DiagnosticsList } from "./designer-shared";

export function ValidationPanel() {
  const diagnostics = useTemplateDesignerStore((state) => state.diagnostics);
  const selectDiagnostic = useSelectDiagnostic();
  const { validate, isValidating, canValidate } = useTemplateDesignerValidationAction();

  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)] overflow-hidden">
      <div className="flex items-center justify-between border-b p-3">
        <div>
          <div className="text-sm font-semibold">Validation Diagnostics</div>
          <div className="text-xs text-muted-foreground">
            {diagnostics.length} diagnostics returned by backend validation
          </div>
        </div>
        <Button
          type="button"
          variant="outline"
          onClick={validate}
          isLoading={isValidating}
          disabled={!canValidate}
        >
          <ListChecksIcon className="size-4" />
          Run
        </Button>
      </div>
      <ScrollArea className="min-h-0" viewportClassName="min-h-0">
        <DiagnosticsList diagnostics={diagnostics} onSelect={selectDiagnostic} />
      </ScrollArea>
    </div>
  );
}
