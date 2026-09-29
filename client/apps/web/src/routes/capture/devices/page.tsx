import { DeviceList } from "@/components/capture/device-list";
import { CaptureDownloadPanel } from "@/components/capture/download-panel";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SectionPanel } from "@/components/section-panel";
import { revokeMyCaptureDevice, type CaptureDevice } from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { KeyRoundIcon, MonitorIcon } from "lucide-react";
import { Link } from "react-router";
import { toast } from "sonner";

export const prefetch: RoutePrefetch = () => [
  queries.capture.myDevices(null),
  queries.capture.access(),
  queries.capture.agentRelease(),
];

type RevokeVariables = { device: CaptureDevice; reason: string };

function AccessNotice() {
  const t = useT();
  const { data: access } = useQuery(queries.capture.access());

  if (access === undefined) {
    return null;
  }
  if (!access.enabled) {
    return (
      <Alert variant="warning" size="sm">
        <AlertDescription>
          {t(
            "Scanning and printing into Trenova is turned off for your organization. Paired computers wait until it is turned on.",
          )}
        </AlertDescription>
      </Alert>
    );
  }
  if (!access.canCapture) {
    return (
      <Alert variant="warning" size="sm">
        <AlertDescription>
          {t(
            "You do not have permission to scan into Trenova. Your paired computers cannot upload until an administrator gives it to you.",
          )}
        </AlertDescription>
      </Alert>
    );
  }

  return null;
}

/**
 * The computers a person has paired, and a way to stop any of them. Anyone
 * signed in can see and revoke their own; nobody else's are here.
 */
export function CaptureDevicesPage() {
  const t = useT();
  const queryClient = useQueryClient();
  const devicesQuery = useQuery(queries.capture.myDevices(null));

  // Failures are shown by the revoke dialog, which stays open until this
  // settles; a toast as well would say the same thing twice.
  const revoke = useMutation({
    mutationFn: ({ device, reason }: RevokeVariables) =>
      revokeMyCaptureDevice(device.id, reason === "" ? null : reason),
    onSuccess: async (device) => {
      toast.success(t("{0} is revoked", device.name));
      await queryClient.invalidateQueries({ queryKey: queries.capture.myDevices._def });
    },
  });

  const devices = devicesQuery.data ?? [];
  const activeCount = devices.filter((device) => device.status === "Active").length;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("My scanners"),
        description: t(
          "The computers you paired with Trenova Capture to scan and print into Trenova",
        ),
        actions: (
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={<Link to="/capture/pair" />}
          >
            <KeyRoundIcon className="size-3.5" aria-hidden />
            {t("Enter a pairing code")}
          </Button>
        ),
      }}
    >
      <AccessNotice />

      <CaptureDownloadPanel />

      <SectionPanel
        title={t("Paired computers")}
        icon={<MonitorIcon aria-hidden />}
        count={activeCount}
      >
        {devicesQuery.isLoading ? (
          <div className="flex flex-col gap-2 p-3" aria-busy="true">
            <Skeleton className="h-14 w-full" />
            <Skeleton className="h-14 w-full" />
          </div>
        ) : devicesQuery.isError && devicesQuery.data === undefined ? (
          <div className="p-3">
            <Alert variant="destructive" size="sm">
              <AlertDescription>{t("Your computers could not be loaded.")}</AlertDescription>
              <AlertAction>
                <Button
                  type="button"
                  size="xs"
                  variant="outline"
                  onClick={() => void devicesQuery.refetch()}
                  isLoading={devicesQuery.isRefetching}
                >
                  {t("Try again")}
                </Button>
              </AlertAction>
            </Alert>
          </div>
        ) : devices.length === 0 ? (
          <EmptySheet
            title={t("No computers paired yet")}
            description={t(
              "Install Trenova Capture on the computer your scanner is plugged into, choose Sign in from its tray icon, and approve the code it shows.",
            )}
            action={
              <Button size="sm" nativeButton={false} render={<Link to="/capture/pair" />}>
                {t("Enter a pairing code")}
              </Button>
            }
            sketch={
              <div className="flex flex-col gap-3 px-6">
                {[0, 1].map((index) => (
                  <div key={index} className="flex flex-col gap-1.5">
                    <GhostLine className="w-1/3" />
                    <GhostLine className="w-2/3" />
                  </div>
                ))}
              </div>
            }
          />
        ) : (
          <DeviceList
            devices={devices}
            showOwner={false}
            canRevoke
            bordered={false}
            onRevoke={(device, reason) => revoke.mutateAsync({ device, reason })}
            revokingId={revoke.isPending ? revoke.variables?.device.id : undefined}
          />
        )}
      </SectionPanel>
    </PageLayout>
  );
}
