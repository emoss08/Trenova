import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { ImpactSummary } from "./impact-summary";

const BACKTEST_LIMIT = 100;

/**
 * The editor's formula run over recent shipments next to what they are
 * charged now: the same impact block the reviewer sees, with the full
 * per-shipment table one click away.
 */
export function BacktestDialog({
  open,
  onOpenChange,
  template,
  source,
  onOpenDetails,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: FormulaTemplate;
  source: string;
  onOpenDetails: () => void;
}) {
  const t = useT();
  const backtest = useQuery({
    queryKey: ["formula-template-backtest", template.id, source],
    queryFn: () =>
      apiService.formulaTemplateService.backtest(template.id ?? "", {
        source,
        limit: BACKTEST_LIMIT,
      }),
    enabled: open && !!template.id && source.trim() !== "",
    staleTime: 30_000,
    retry: false,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Backtest")}</DialogTitle>
          <DialogDescription>
            {backtest.data
              ? t(
                  "Compares what {0} recent shipments billed with what the editor's content would charge them.",
                  backtest.data.summary.shipmentCount,
                )
              : t(
                  "Compares what recent shipments billed with what the editor's content would charge them.",
                )}
          </DialogDescription>
        </DialogHeader>
        <ImpactSummary
          data={backtest.data}
          isLoading={backtest.isPending}
          isError={backtest.isError}
          showFailures
        />
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              onOpenChange(false);
              onOpenDetails();
            }}
          >
            {t("Every shipment")}
          </Button>
          <Button type="button" onClick={() => onOpenChange(false)}>
            {t("Done")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
