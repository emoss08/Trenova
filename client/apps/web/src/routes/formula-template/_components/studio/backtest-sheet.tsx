import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import type {
  FormulaTemplate,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import { HistoryIcon } from "lucide-react";
import { lazy, Suspense } from "react";
import type { UseFormReturn } from "react-hook-form";

const FormulaTemplateBacktestTab = lazy(() => import("../formula-template-backtest-tab"));

type BacktestSheetProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  form: UseFormReturn<FormulaTemplateFormValues>;
  template: FormulaTemplate | null;
};

export function BacktestSheet({ open, onOpenChange, form, template }: BacktestSheetProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="flex w-[620px] flex-col gap-0 sm:max-w-[620px]">
        <SheetHeader className="border-b pb-3">
          <SheetTitle className="flex items-center gap-2">
            <HistoryIcon className="size-4" />
            Backtest
          </SheetTitle>
          <SheetDescription>
            Re-rate recent shipments with a candidate expression and compare against what they
            charge today.
          </SheetDescription>
        </SheetHeader>
        <ScrollArea className="min-h-0 flex-1">
          <div className="p-4">
            <Suspense fallback={null}>
              <FormulaTemplateBacktestTab form={form} template={template} />
            </Suspense>
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}
