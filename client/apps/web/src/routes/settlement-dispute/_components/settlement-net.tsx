import { AmountDisplay } from "@trenova/shared/components/accounting/amount-display";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * The disputed settlement's net pay. The settlement is linked from the dispute, and its pay is
 * outside its reference view: without driver settlement read it is withheld.
 */
export function SettlementNet({
  netPayMinor,
  currencyCode,
}: {
  netPayMinor: number | null;
  currencyCode: string | null;
}) {
  const t = useT();
  return (
    <span className="inline-flex items-center gap-1">
      {t("Net")}
      {netPayMinor === null || currencyCode === null ? (
        <WithheldValue kind="driver_settlement" />
      ) : (
        <AmountDisplay value={netPayMinor} currency={currencyCode} />
      )}
    </span>
  );
}
