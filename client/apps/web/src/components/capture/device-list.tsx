import type { CaptureDevice, CaptureFleetDevice } from "@/lib/graphql/capture";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo, formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { MonitorIcon, UnplugIcon } from "lucide-react";
import { useId, useState } from "react";

type AnyDevice = CaptureDevice | (Partial<Pick<CaptureFleetDevice, "user">> & CaptureDevice);

function Presence({ device, now }: { device: CaptureDevice; now: number }) {
  const t = useT();

  if (device.status === "Revoked") {
    return <Badge variant="neutral">{t("Revoked")}</Badge>;
  }
  if (device.isOnline) {
    return <Badge variant="success">{t("Online")}</Badge>;
  }

  return (
    <span className="text-foreground-subtle text-xs">
      {device.lastSeenAt === null
        ? t("Never connected")
        : t("Last seen {0}", formatSecondsAgo(Math.max(0, now - device.lastSeenAt)))}
    </span>
  );
}

/**
 * The computers running Trenova Capture, each with what it can reach. A
 * revoked computer stays on the list, so a person can see it was theirs and
 * when it stopped.
 */
export function DeviceList({
  devices,
  now,
  showOwner,
  canRevoke,
  onRevoke,
  revokingId,
}: {
  devices: AnyDevice[];
  now: number;
  showOwner: boolean;
  canRevoke: boolean;
  onRevoke: (device: CaptureDevice, reason: string) => void;
  revokingId: string | undefined;
}) {
  const t = useT();
  const [revoking, setRevoking] = useState<CaptureDevice | null>(null);

  return (
    <>
      <ul className="border-border divide-border-subtle bg-card divide-y rounded-lg border">
        {devices.map((device) => (
          <li key={device.id} className="flex flex-col gap-2 px-3 py-3 sm:flex-row sm:items-start">
            <span
              className={cn(
                "bg-sunken flex size-8 shrink-0 items-center justify-center rounded-md",
                device.status === "Revoked" ? "text-foreground-subtle" : "text-foreground-muted",
              )}
            >
              <MonitorIcon className="size-4" aria-hidden />
            </span>
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-sm font-medium">{device.name}</span>
                <Presence device={device} now={now} />
              </div>
              <p className="text-foreground-muted text-xs">
                {[
                  showOwner && "user" in device && device.user ? device.user.name : null,
                  device.machineName,
                  device.windowsUser === "" ? null : device.windowsUser,
                  t("Trenova Capture {0}", device.agentVersion),
                  device.osVersion === "" ? null : device.osVersion,
                ]
                  .filter(Boolean)
                  .join(" · ")}
              </p>
              {device.status === "Revoked" ? (
                <p className="text-foreground-subtle text-xs">
                  {device.revokedAt !== null &&
                    t("Revoked {0}", formatUnixDateTime(device.revokedAt))}
                  {device.revokedReason !== "" && ` · ${device.revokedReason}`}
                </p>
              ) : device.sources.length > 0 ? (
                <p className="text-foreground-subtle text-xs">
                  {t("Scanners: {0}", device.sources.map((source) => source.name).join(", "))}
                </p>
              ) : (
                <p className="text-foreground-subtle text-xs">{t("No scanner reported yet")}</p>
              )}
            </div>
            {canRevoke && device.status === "Active" && (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setRevoking(device)}
                isLoading={revokingId === device.id}
              >
                <UnplugIcon className="size-3.5" />
                {t("Revoke")}
              </Button>
            )}
          </li>
        ))}
      </ul>

      <RevokeDeviceDialog
        device={revoking}
        onCancel={() => setRevoking(null)}
        onConfirm={(reason) => {
          if (revoking !== null) {
            onRevoke(revoking, reason);
          }
          setRevoking(null);
        }}
      />
    </>
  );
}

function RevokeDeviceDialog({
  device,
  onCancel,
  onConfirm,
}: {
  device: CaptureDevice | null;
  onCancel: () => void;
  onConfirm: (reason: string) => void;
}) {
  const t = useT();
  const reasonId = useId();
  const [reason, setReason] = useState("");

  return (
    <AlertDialog
      open={device !== null}
      onOpenChange={(open) => {
        if (!open) {
          setReason("");
          onCancel();
        }
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-danger-subtle text-destructive">
            <UnplugIcon />
          </AlertDialogMedia>
          <AlertDialogTitle>{t("Revoke {0}?", device?.name ?? "")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "It stops working at once, including a scan it is in the middle of. Pages it already uploaded stay in Intake. To use it again, pair it again.",
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <div className="flex flex-col gap-1">
          <label htmlFor={reasonId} className="text-foreground-subtle text-xs font-medium">
            {t("Why")}
          </label>
          <Input
            id={reasonId}
            value={reason}
            maxLength={255}
            placeholder={t("Optional, for the audit trail")}
            onChange={(event) => setReason(event.target.value)}
          />
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("Keep it")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => {
              onConfirm(reason.trim());
              setReason("");
            }}
          >
            {t("Revoke")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
