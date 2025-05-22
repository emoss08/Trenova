import { upperFirst } from "@/lib/utils";
import type { Resource } from "@/types/audit-entry";
import type { Action } from "@/types/roles-permissions";
import { Button } from "./button";

export function PermissionTextSkeleton({
  resource,
  action,
}: {
  resource: Resource;
  action: Action;
}) {
  return (
    <p className="text-xs text-destructive">
      You do not have permissions to {action} {resource}s
    </p>
  );
}

export function DataTablePermissionDeniedSkeleton({
  resource,
  action,
}: {
  resource: Resource;
  action: Action;
}) {
  // TODO(wolfred): Once the notification system is added we need to add a handler to send a notification to the admin
  return (
    <div className="w-full min-h-[calc(100vh-30rem)] bg-muted/50 flex items-center justify-center border border-dashed border-border rounded-md p-8 text-center">
      <div className="flex flex-col items-center gap-4">
        <div className="flex flex-col gap-1">
          <p className="text-2xl font-medium text-foreground">Access Denied</p>
          <p className="text-base text-muted-foreground">
            You don&apos;t have permission to {action} {upperFirst(resource)}.
          </p>
        </div>
        <p className="text-sm text-muted-foreground max-w-md">
          To gain access, please contact your system administrator and request
          the necessary permissions for the &quot;{upperFirst(resource)}&quot;
          resource.
        </p>
        <Button size="sm">Request Access</Button>
      </div>
    </div>
  );
}
