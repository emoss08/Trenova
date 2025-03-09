import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { Icon } from "@/components/ui/icons";
import { faExclamationTriangle } from "@fortawesome/pro-regular-svg-icons";
import { lazy, memo } from "react";

const AuditLogTable = lazy(() => import("./_components/audit-log-table"));

export function AuditLogs() {
  return (
    <div className="flex flex-col space-y-6">
      <MetaTags
        title="Audit Entries"
        description="View and manage audit entries"
      />
      <Header />
      <AuditAlert />
      <SuspenseLoader>
        <AuditLogTable />
      </SuspenseLoader>
    </div>
  );
}

function Header() {
  return (
    <div className="flex justify-between items-center">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Audit Entries</h1>
        <p className="text-muted-foreground">
          Monitor, track, and analyze system activities and user actions
        </p>
      </div>
    </div>
  );
}

const AuditAlert = memo(() => {
  return (
    <div className="flex bg-red-500/20 border border-red-600/50 p-4 rounded-md justify-between items-center mb-4 w-full">
      <div className="flex items-center gap-2 w-full text-red-600">
        <Icon icon={faExclamationTriangle} className="size-4" />
        <div className="flex flex-col">
          <p className="text-sm font-medium">Audit Logs Processing</p>
          <p className="text-xs dark:text-red-100">
            Audit logs are processed in batches and may take a few moments to
            appear. If logs are not immediately visible, please refresh the page
            after a brief wait.
          </p>
        </div>
      </div>
    </div>
  );
});
AuditAlert.displayName = "AuditAlert";
