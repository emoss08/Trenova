import { CaptureDownloadPanel } from "@/components/capture/download-panel";
import { DeviceList } from "@/components/capture/device-list";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { revokeMyCaptureDevice, type CaptureDevice } from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { KeyRoundIcon } from "lucide-react";
import { Link } from "react-router";
import { toast } from "sonner";

export const prefetch: RoutePrefetch = () => [
  queries.capture.myDevices(null),
  queries.capture.access(),
  queries.capture.agentRelease(),
];

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * The computers a person has paired, and a way to stop any of them. Anyone
 * signed in can see and revoke their own; nobody else's are here.
 */
export function CaptureDevicesPage() {
  const t = useT();
  const queryClient = useQueryClient();
  const devicesQuery = useQuery(queries.capture.myDevices(null));
  const accessQuery = useQuery(queries.capture.access());
  const now = nowInSeconds();

  const revoke = useApiMutation({
    mutationFn: ({ device, reason }: { device: CaptureDevice; reason: string }) =>
      revokeMyCaptureDevice(device.id, reason === "" ? null : reason),
    onSuccess: async (device) => {
      toast.success(t("{0} is revoked", device.name));
      await queryClient.invalidateQueries({ queryKey: queries.capture.myDevices._def });
    },
    resourceName: "Device",
  });

  const devices = devicesQuery.data ?? [];
  const access = accessQuery.data;

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
            <KeyRoundIcon className="size-3.5" />
            {t("Enter a pairing code")}
          </Button>
        ),
      }}
    >
      {access !== undefined && !access.enabled && (
        <Alert variant="warning" size="sm">
          <AlertDescription>
            {t(
              "Scanning and printing into Trenova is turned off for your organization. Paired computers wait until it is turned on.",
            )}
          </AlertDescription>
        </Alert>
      )}

      <CaptureDownloadPanel />

      {devicesQuery.isLoading ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      ) : devicesQuery.isError ? (
        <Alert variant="destructive" size="sm">
          <AlertDescription>{t("Your computers could not be loaded.")}</AlertDescription>
        </Alert>
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
                <div key={index} className="flex gap-3">
                  <div className="bg-sunken size-8 shrink-0 rounded-md" />
                  <div className="flex flex-1 flex-col gap-1.5">
                    <GhostLine className="w-1/3" />
                    <GhostLine className="w-2/3" />
                  </div>
                </div>
              ))}
            </div>
          }
        />
      ) : (
        <DeviceList
          devices={devices}
          now={now}
          showOwner={false}
          canRevoke
          onRevoke={(device, reason) => revoke.mutate({ device, reason })}
          revokingId={revoke.isPending ? revoke.variables?.device.id : undefined}
        />
      )}
    </PageLayout>
  );
}
