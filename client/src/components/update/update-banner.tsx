import { usePermissionStore } from "@/stores/permission-store";
import { useUpdateStore } from "@/stores/update-store";
import { ArrowUpCircleIcon, ExternalLinkIcon, XIcon } from "lucide-react";
import { Button } from "../ui/button";

export function UpdateBanner() {
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
    <div className="group relative flex flex-col items-center gap-2 rounded-md border border-info/30 bg-info/4 px-4 py-2">
      <button
        onClick={handleDismiss}
        className="absolute -top-2 -right-2 cursor-pointer rounded-full bg-foreground p-0.5 text-background opacity-0 shadow-sm transition-opacity group-hover:opacity-100"
      >
        <XIcon className="size-4" />
      </button>
      <div className="flex items-center gap-2 text-purple-600 dark:text-purple-400">
        <ArrowUpCircleIcon className="size-5" />
        <span className="text-sm font-semibold">
          Update available: v{status.latestVersion}
        </span>
      </div>
      <span className="text-center text-xs text-purple-600 dark:text-purple-50">
        You are currently running v{status.currentVersion}
      </span>
      <div className="flex items-center gap-2">
        {status.latestRelease.htmlUrl && (
          <Button
            variant="ghost"
            size="sm"
            className="text-purple-600 hover:bg-purple-500/20 hover:text-purple-700 dark:text-purple-400 dark:hover:bg-purple-500/20 dark:hover:text-purple-300"
            onClick={() => window.open(status.latestRelease?.htmlUrl, "_blank")}
          >
            View Release
            <ExternalLinkIcon className="size-3" />
          </Button>
        )}
      </div>
    </div>
  );
}
