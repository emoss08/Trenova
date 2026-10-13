import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { Dataflow02Icon, PlusIcon } from "@trenova/shared/components/icons";
import { useMemo } from "react";
import { useFormContext } from "react-hook-form";
import {
  BreakdownDefinitionEditor,
  type BreakdownPricing,
} from "../../breakdown-definition-editor";
import { previewAmounts } from "../model/format";
import { useStudio } from "../studio-context";
import { SectionHead } from "./section-head";

export function BreakdownSection() {
  const t = useT();
  const { known, preview, reconciliation } = useStudio();
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const result = preview.result;

  const pricing = useMemo((): BreakdownPricing | null => {
    if (!result?.valid) return null;
    const amounts = previewAmounts(result);
    return {
      lines: new Map((result.breakdown ?? []).map((line) => [line.name, line])),
      reconciliation,
      clamped:
        amounts?.bound && amounts.bound !== null
          ? { bound: amounts.bound, billed: amounts.billed }
          : null,
    };
  }, [result, reconciliation]);

  return (
    <section className="flex flex-col gap-3">
      <BreakdownDefinitionEditor
        control={control as never}
        knownIdentifiers={known}
        pricing={pricing}
        head={({ add, atLimit }) => (
          <SectionHead
            section="breakdown"
            icon={Dataflow02Icon}
            title={t("Breakdown")}
            description={t(
              "Itemize the total into named lines for invoices and disputes. Lines should add up to the formula's result.",
            )}
          >
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="gap-1"
              disabled={atLimit}
              onClick={add}
            >
              <PlusIcon className="size-3" />
              {t("Add line")}
            </Button>
          </SectionHead>
        )}
      />
    </section>
  );
}
