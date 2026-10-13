import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Download01Icon, Mail01Icon } from "@trenova/shared/components/icons";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import type { StatementRange } from "@/types/customer-statement";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { downloadBlob } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { SendStatementDialog } from "./send-statement-dialog";

/**
 * Download the statement on screen as the PDF a customer receives, or email
 * it to them. Emailing works the account, so it needs the update permission
 * on accounts receivable.
 */
export function StatementActions({
  customerId,
  customerName,
  range,
  rangeLabel,
}: {
  customerId: string;
  customerName: string;
  range: StatementRange;
  rangeLabel: string;
}) {
  const t = useT();
  const [sendOpen, setSendOpen] = useState(false);
  const canSend = usePermissionStore((state) =>
    state.hasPermission(Resource.AccountsReceivable, Operation.Update),
  );

  const { mutate: download, isPending: isDownloading } = useApiMutation({
    mutationFn: () => apiService.customerStatementService.downloadPdf(customerId, range),
    resourceName: "CustomerStatement",
    onSuccess: ({ blob, fileName }) => downloadBlob(blob, fileName),
  });

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        isLoading={isDownloading}
        loadingText={t("Preparing PDF...")}
        onClick={() => download()}
      >
        <Download01Icon className="size-3.5" />
        {t("Download PDF")}
      </Button>
      {canSend ? (
        <Button size="sm" onClick={() => setSendOpen(true)}>
          <Mail01Icon className="size-3.5" />
          {t("Email statement")}
        </Button>
      ) : null}
      <SendStatementDialog
        customerId={customerId}
        customerName={customerName}
        range={range}
        rangeLabel={rangeLabel}
        open={sendOpen}
        onOpenChange={setSendOpen}
      />
    </>
  );
}
