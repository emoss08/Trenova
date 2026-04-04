import { cn } from "@/lib/utils";
import { usePermissionStore } from "@/stores/permission-store";
import { useUpdateStore } from "@/stores/update-store";
import { ExternalLinkIcon, XIcon } from "lucide-react";
import { Button } from "../ui/button";

export function LatestChange() {
  const manifest = usePermissionStore((state) => state.manifest);
  const status = useUpdateStore((state) => state.status);
  const dismissedVersion = useUpdateStore((state) => state.dismissedVersion);
  const dismissUpdate = useUpdateStore((state) => state.dismissUpdate);

  const isAdmin = manifest?.isPlatformAdmin || manifest?.isOrgAdmin;

  if (!isAdmin) {
    return null;
  }

  if (!status?.updateAvailable || !status.latestRelease) {
    return null;
  }

  if (dismissedVersion === status.latestVersion) {
    return null;
  }

  const handleDismiss = () => {
    if (status.latestVersion) {
      dismissUpdate(status.latestVersion);
    }
  };

  return (
    <div
      className={cn(
        "group/latest-change size-full min-h-27 justify-center border-t",
        "relative flex size-full flex-col gap-1 overflow-hidden px-4 pt-3 pb-1 *:text-nowrap",
        "transition-opacity group-data-[collapsible=icon]:pointer-events-none group-data-[collapsible=icon]:opacity-0",
      )}
    >
      <span className="font-light font-mono text-[10px] text-muted-foreground">UPDATE</span>
      <p className="font-medium text-xs">v{status.latestVersion} available</p>
      <span className="text-[10px] text-muted-foreground">Running v{status.currentVersion}</span>
      {status.latestRelease.htmlUrl && (
        <Button
          render={
            <a href={status.latestRelease.htmlUrl} target="_blank" rel="noopener noreferrer">
              View Release
            </a>
          }
          className="w-max px-0 font-light text-xs"
          size="sm"
          variant="link"
        >
          <ExternalLinkIcon className="size-3" />
        </Button>
      )}
      <Button
        className="absolute top-2 right-2 z-10 size-6 rounded-full opacity-0 transition-opacity group-hover/latest-change:opacity-100"
        onClick={handleDismiss}
        size="icon-sm"
        variant="ghost"
      >
        <XIcon className="size-3.5 text-muted-foreground" />
      </Button>
    </div>
  );
}
