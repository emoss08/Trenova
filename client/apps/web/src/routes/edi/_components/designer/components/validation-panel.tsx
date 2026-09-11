import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

  const diagnostics = useTemplateDesignerStore((state) => state.diagnostics);
  const selectDiagnostic = useSelectDiagnostic();
  const { validate, isValidating, canValidate } = useTemplateDesignerValidationAction();

  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)] overflow-hidden">
      <div className="flex items-center justify-between border-b p-3">
        <div>
          <div className="text-sm font-semibold">{t("Validation Diagnostics")}</div>
          <div className="text-muted-foreground text-xs">
            {t("{0} diagnostics returned by backend validation", diagnostics.length)}
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
          {t("Run")}
        </Button>
      </div>
      <ScrollArea className="min-h-0" viewportClassName="min-h-0">
        <DiagnosticsList diagnostics={diagnostics} onSelect={selectDiagnostic} />
      </ScrollArea>
    </div>
  );
}
