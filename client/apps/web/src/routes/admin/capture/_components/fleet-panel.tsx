import { DeviceList } from "@/components/capture/device-list";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  revokeCaptureDevice,
  type CaptureDevice,
  type CaptureDeviceStatus,
} from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Input } from "@trenova/shared/components/ui/input";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { SearchIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

type StatusFilter = "Active" | "Revoked" | "all";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/** Every computer paired in the organization, and a way to stop any of them. */
export function FleetPanel() {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canRevoke } = usePermission(Resource.CaptureDevice, Operation.Update);
  const [status, setStatus] = useState<StatusFilter>("Active");
  const [search, setSearch] = useState("");
  const query = useDebounce(search.trim(), 250);
  const devicesQuery = useQuery(
    queries.capture.devices(status === "all" ? null : (status as CaptureDeviceStatus), query),
  );

  const revoke = useApiMutation({
    mutationFn: ({ device, reason }: { device: CaptureDevice; reason: string }) =>
      revokeCaptureDevice(device.id, reason === "" ? null : reason),
    onSuccess: async (device) => {
      toast.success(t("{0} is revoked", device.name));
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.capture.devices._def }),
        queryClient.invalidateQueries({ queryKey: queries.capture.myDevices._def }),
      ]);
    },
    resourceName: "Device",
  });

  const devices = devicesQuery.data ?? [];

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <SegmentedControl<StatusFilter>
          aria-label={t("Which computers")}
          value={status}
          onValueChange={setStatus}
          items={[
            { value: "Active", label: t("Paired") },
            { value: "Revoked", label: t("Revoked") },
            { value: "all", label: t("All") },
          ]}
        />
        <Input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t("Search computer, user or Windows account")}
          aria-label={t("Search computer, user or Windows account")}
          className="max-w-sm"
          leftElement={<SearchIcon className="text-foreground-subtle size-3.5" />}
          rightElement={
            search === "" ? undefined : (
              <Button
                size="icon-xs"
                variant="ghost"
                aria-label={t("Clear the search")}
                onClick={() => setSearch("")}
              >
                <XIcon className="size-3.5" />
              </Button>
            )
          }
        />
      </div>

      {devicesQuery.isLoading ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton className="h-16 w-full" />
          <Skeleton className="h-16 w-full" />
        </div>
      ) : devicesQuery.isError ? (
        <Alert variant="destructive" size="sm">
          <AlertDescription>{t("Computers could not be loaded.")}</AlertDescription>
        </Alert>
      ) : devices.length === 0 ? (
        <EmptySheet
          title={query === "" ? t("No computers paired") : t("No computer matches")}
          description={t(
            "A computer appears here once somebody approves it from Trenova Capture's pairing code.",
          )}
          sketch={
            <div className="flex flex-col gap-2 px-6">
              <GhostLine className="w-1/3" />
              <GhostLine className="w-2/3" />
            </div>
          }
        />
      ) : (
        <DeviceList
          devices={devices}
          now={nowInSeconds()}
          showOwner
          canRevoke={canRevoke}
          onRevoke={(device, reason) => revoke.mutate({ device, reason })}
          revokingId={revoke.isPending ? revoke.variables?.device.id : undefined}
        />
      )}
    </div>
  );
}
