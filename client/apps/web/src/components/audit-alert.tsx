import { useT } from "@trenova/shared/i18n/use-t";
import { AlertTriangleIcon } from "@trenova/shared/components/icons";

export function AuditAlert() {
  const t = useT();

  return (
    <div className="flex w-full items-center justify-between rounded-md border border-danger-border bg-danger-subtle p-4">
      <div className="flex w-full items-center gap-3 text-danger-foreground">
        <AlertTriangleIcon className="size-5 shrink-0" />
        <div className="flex flex-col">
          <p className="text-sm font-medium">{t("Audit logs processing")}</p>
          <p className="text-xs dark:text-danger-foreground">
            {t(
              "Audit logs are processed in batches and may take a few moments to appear. If logs are not immediately visible, please refresh the page after a brief wait.",
            )}
          </p>
        </div>
      </div>
    </div>
  );
}
