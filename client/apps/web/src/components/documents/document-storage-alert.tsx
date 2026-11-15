import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { formatFileSize } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { usePermission } from "@/hooks/use-permission";
import { documentControl } from "@/lib/queries/document-control";
import { useQuery } from "@tanstack/react-query";

export function DocumentStorageAlert({ className }: { className?: string }) {
  const t = useT();
  const { allowed: isAdministrator } = usePermission(Resource.Organization, Operation.Update);
  const { data } = useQuery({
    ...documentControl.storageStatus(),
    enabled: isAdministrator,
  });

  if (!isAdministrator || !data?.managed || (!data.warning && !data.stopped)) {
    return null;
  }

  const used = formatFileSize(data.used);
  const included = formatFileSize(data.included);

  if (data.stopped) {
    return (
      <Alert variant="destructive" size="sm" className={className}>
        <AlertCircleIcon />
        <AlertTitle>{t("Document storage is full")}</AlertTitle>
        <AlertDescription>
          <p>
            {t(
              "{0} of the {1} your plan includes is used, so uploads are stopped.",
              used,
              included,
            )}
          </p>
          <p>
            {t(
              "Turn on overage from your Trenova account to keep uploading, or delete documents you no longer need.",
            )}
          </p>
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <Alert variant="warning" size="sm" className={className}>
      <AlertCircleIcon />
      <AlertTitle>{t("Document storage is nearly full")}</AlertTitle>
      <AlertDescription>
        <p>{t("{0} of the {1} your plan includes is used.", used, included)}</p>
        <p>
          {data.overage
            ? t("Past the included amount, storage is billed as overage from your Trenova account.")
            : t(
                "When it is full, uploads stop. Turn on overage from your Trenova account, or delete documents you no longer need.",
              )}
        </p>
      </AlertDescription>
    </Alert>
  );
}
