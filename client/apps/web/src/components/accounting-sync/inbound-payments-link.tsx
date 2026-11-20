import { Link } from "@trenova/shared/components/link";
import { ACCOUNTING_INBOUND_PATH } from "@/lib/accounting-sync";
import { accountingSync } from "@/lib/queries/accounting-sync";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@trenova/shared/types/permission";

export function InboundPaymentsLink({ system }: { system: AccountingSystem }) {
  const t = useT();
  const { allowed } = usePermission(Resource.AccountingReconciliation, Operation.Read);
  const overview = useQuery({ ...accountingSync.inboundOverview(system), enabled: allowed });
  const data = overview.data;
  if (!allowed || !data) {
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
