import { ACCOUNTING_INBOUND_PATH } from "@/lib/accounting-sync";
import { queries } from "@/lib/queries";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link } from "react-router";

export function InboundPaymentsLink({ system }: { system: AccountingSystem }) {
  const t = useT();
  const overview = useQuery(queries.accountingSync.inboundOverview(system));
  const data = overview.data;
  if (!data) {
    return null;
  }
  const waiting = data.summary.proposed;
  if (data.policy === "Off" && waiting === 0) {
    return null;
  }

  return (
    <Button
      type="button"
      size="sm"
      variant="outline"
      render={<Link to={ACCOUNTING_INBOUND_PATH} />}
    >
      {waiting > 0 ? t("Payments to apply ({0})", waiting) : t("Payments from the books")}
    </Button>
  );
}
