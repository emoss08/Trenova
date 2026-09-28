import { CaptureDownloadPanel } from "@/components/capture/download-panel";
import { DeviceList } from "@/components/capture/device-list";
import { usePermission } from "@/hooks/use-permission";
import {
  revokeCaptureDevice,
  type CaptureDevice,
  type CaptureDeviceStatus,
} from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { keepPreviousData, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Input } from "@trenova/shared/components/ui/input";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { SearchIcon, XIcon } from "lucide-react";
import { useState, type ReactNode } from "react";
import { toast } from "sonner";

type StatusFilter = "Active" | "Revoked" | "all";

function FleetEmpty({
  status,
  query,
  onClearSearch,
}: {
  status: StatusFilter;
  query: string;
  onClearSearch: () => void;
}) {
  const t = useT();
  const sketch = (
    <div className="flex flex-col gap-2 px-6">
      <GhostLine className="w-1/3" />
      <GhostLine className="w-2/3" />
    </div>
  );

  let title: string;
  let description: string;
  let action: ReactNode = undefined;
  if (query !== "") {
    title = t("No computer matches");
    description = t("Check the spelling, or clear the search to see every computer.");
    action = (
      <Button size="sm" variant="outline" onClick={onClearSearch}>
        {t("Clear the search")}
      </Button>
    );
  } else if (status === "Revoked") {
    title = t("No revoked computers");
    description = t("A revoked computer stops scanning and stays listed here.");
  } else {
    title = t("No computers paired");
    description = t(
      "A computer appears here once somebody approves it from Trenova Capture's pairing code.",
    );
  }

  return <EmptySheet title={title} description={description} action={action} sketch={sketch} />;
}

/** Every computer paired in the organization, and a way to stop any of them. */
export function FleetPanel() {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canRevoke } = usePermission(Resource.CaptureDevice, Operation.Update);
  const [status, setStatus] = useState<StatusFilter>("Active");
  const [search, setSearch] = useState("");
  const query = useDebounce(search.trim(), 250);
  // The last list stays up while the next filter or search is fetched, so
  // typing does not flash the list away on every pause.
  const devicesQuery = useQuery({
    ...queries.capture.devices(status === "all" ? null : (status as CaptureDeviceStatus), query),
    placeholderData: keepPreviousData,
  });

  // Failures are shown by the revoke dialog, which stays open until this
  // settles; a toast as well would say the same thing twice.
  const revoke = useMutation({
    mutationFn: ({ device, reason }: { device: CaptureDevice; reason: string }) =>
      revokeCaptureDevice(device.id, reason === "" ? null : reason),
    onSuccess: async (device) => {
      toast.success(t("{0} is revoked", device.name));
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.capture.devices._def }),
        queryClient.invalidateQueries({ queryKey: queries.capture.myDevices._def }),
      ]);
    },
  });

  const devices = devicesQuery.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      <CaptureDownloadPanel
        whenMissing={
          <Alert variant="info" size="sm">
            <AlertDescription>
              {t(
                "No Trenova Capture installer is published yet, or this server does not serve one. Once it is, people download it from here and from My scanners.",
              )}
            </AlertDescription>
          </Alert>
        }
      />

      <section aria-label={t("Computers")} className="flex flex-col gap-3">
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
            className="w-full sm:max-w-sm"
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

        {devicesQuery.isPending ? (
          <div className="flex flex-col gap-2" aria-busy="true">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : devicesQuery.isError ? (
          <ErrorState
            error={devicesQuery.error}
            layout="compact"
            title={t("Computers could not be loaded.")}
            onRetry={() => void devicesQuery.refetch()}
          />
        ) : devices.length === 0 ? (
          <FleetEmpty status={status} query={query} onClearSearch={() => setSearch("")} />
        ) : (
          <div
            className={cn(
              "transition-opacity duration-150",
              devicesQuery.isPlaceholderData && "opacity-60",
            )}
          >
            <DeviceList
              devices={devices}
              showOwner
              canRevoke={canRevoke}
              onRevoke={(device, reason) => revoke.mutateAsync({ device, reason })}
              revokingId={revoke.isPending ? revoke.variables?.device.id : undefined}
            />
          </div>
        )}
      </section>
    </div>
  );
}
