import { useT } from "@trenova/shared/i18n/use-t";
import { TriangleAlert } from "lucide-react";

export function AuditAlert() {
  const t = useT();

  return (
    <div className="flex w-full items-center justify-between rounded-md border border-red-600/50 bg-red-500/10 p-4">
      <div className="flex w-full items-center gap-3 text-red-600">
        <TriangleAlert className="size-5 shrink-0" />
        <div className="flex flex-col">
          <p className="text-sm font-medium">{t("Audit Logs Processing")}</p>
          <p className="text-xs dark:text-red-100">
            {t("Audit logs are processed in batches and may take a few moments to appear. If logs are not immediately visible, please refresh the page after a brief wait.")}
          </p>
        </div>
      </div>
    </div>
  );
}
