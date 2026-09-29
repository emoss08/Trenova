import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { windowsRequirement } from "@/lib/capture-release";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { APP_ENV } from "@trenova/shared/lib/constants";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { formatFileSize } from "@trenova/shared/lib/utils";
import { DownloadIcon } from "lucide-react";
import { CaptureDevelopmentInstallPanel } from "./development-install-panel";

type CaptureDownloadPanelProps = {
  /**
   * Shown when no release is published: nothing on a person's own page, a
   * word to the administrator on theirs.
   */
  whenMissing?: React.ReactNode;
  /**
   * A development server publishes no release; in its place the panel says
   * how to install a development build. Defaults to the app's own mode.
   */
  development?: boolean;
};

/**
 * The current Trenova Capture installer, as the server's signed manifest
 * describes it. The download is the release's own file; what the page shows
 * (version, size, checksum) is what the manifest promised, so a person who
 * checks the file by hand has something to check it against.
 */
export function CaptureDownloadPanel({
  whenMissing,
  development = APP_ENV === "development",
}: CaptureDownloadPanelProps) {
  const t = useT();
  // A release changes rarely and the server caches it; an hour is fine.
  const releaseQuery = useQuery({ ...queries.capture.agentRelease(), staleTime: 60 * 60 * 1000 });

  if (releaseQuery.isLoading) {
    return <Skeleton className="h-24 w-full" aria-busy="true" />;
  }
  // A failed request says nothing about whether a release exists, so it must
  // not read as "not published": that sends an administrator looking for a
  // missing upload when the fault is the connection.
  if (releaseQuery.isError && releaseQuery.data === undefined) {
    return (
      <Alert variant="destructive" size="sm">
        <AlertDescription>
          {t("The Trenova Capture download could not be loaded.")}
        </AlertDescription>
        <AlertAction>
          <Button
            type="button"
            size="xs"
            variant="outline"
            onClick={() => void releaseQuery.refetch()}
            isLoading={releaseQuery.isRefetching}
          >
            {t("Try again")}
          </Button>
        </AlertAction>
      </Alert>
    );
  }
  const release = releaseQuery.data;
  if (!release) {
    if (development) {
      return <CaptureDevelopmentInstallPanel />;
    }
    return whenMissing ? <>{whenMissing}</> : null;
  }

  const requirement = windowsRequirement(release.minimumWindowsBuild);
  const requires = requirement.name
    ? t("{0} or later, 64-bit", requirement.name)
    : t("Windows build {0} or later, 64-bit", String(requirement.build));

  return (
    <SectionPanel
      title={t("Install Trenova Capture")}
      icon={<DownloadIcon aria-hidden />}
      hint={t("Version {0}", release.version)}
      action={
        <Button
          size="sm"
          nativeButton={false}
          render={
            <a
              href={release.installer.url}
              download={release.installer.fileName}
              rel="noreferrer"
            />
          }
        >
          <DownloadIcon className="size-3.5" aria-hidden />
          {t("Download Trenova Capture")}
        </Button>
      }
    >
      <div className="px-3 py-3">
        <DescriptionList>
          <DescriptionItem label={t("Released")}>
            {formatUnixInUserTimezone(release.publishedAt, { dateStyle: "medium" })}
          </DescriptionItem>
          <DescriptionItem label={t("Size")} numeric>
            {formatFileSize(release.installer.size)}
          </DescriptionItem>
          <DescriptionItem label={t("Requires")}>{requires}</DescriptionItem>
          <DescriptionItem
            label={t("SHA-256")}
            span="full"
            valueClassName="font-mono text-xs break-all"
          >
            {release.installer.sha256}
          </DescriptionItem>
        </DescriptionList>
      </div>
      <SectionPanelQuiet>
        {t(
          "Installing takes administrator rights. It adds the Trenova printer and starts Trenova Capture in the notification area at sign-in; once it is installed, updates arrive by themselves unless your organization turns that off.",
        )}
        {release.notes ? ` ${release.notes}` : null}
      </SectionPanelQuiet>
    </SectionPanel>
  );
}
