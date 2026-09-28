import { useNowSeconds } from "@/hooks/use-now-seconds";
import {
  captureDevicePresence,
  captureDevicePresenceAttrs,
  type CaptureDevicePresence,
} from "@/lib/capture";
import type { CaptureDevice, CaptureFleetDevice } from "@/lib/graphql/capture";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo, formatUnixDateTime } from "@trenova/shared/lib/date";
import { phaseTone, type BadgeAttrProps } from "@trenova/shared/lib/status-phase";
import { cn } from "@trenova/shared/lib/utils";
import { UnplugIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { RevokeDeviceDialog } from "./revoke-device-dialog";

type AnyDevice = CaptureDevice | (Partial<Pick<CaptureFleetDevice, "user">> & CaptureDevice);

/** How often "Last seen" moves on while the list stays open. */
const PRESENCE_TICK_MS = 30_000;

export type DeviceListProps = {
  devices: AnyDevice[];
  /**
   * The time the caller last rendered at. The list keeps its own clock as
   * well and shows whichever is later, so a caller that computes this once
   * per render still gets a "Last seen" that moves on.
   */
  now?: number;
  showOwner: boolean;
  canRevoke: boolean;
  /**
   * Revokes a computer. Return the request's promise and the dialog stays
   * open until it settles, showing why it failed if it does; return nothing
   * and the dialog closes as soon as it is confirmed.
   */
  onRevoke: (device: CaptureDevice, reason: string) => unknown;
  revokingId: string | undefined;
  /** Drop the list's own border when it sits inside a panel that has one. */
  bordered?: boolean;
};

function Presence({
  device,
  now,
  attrs,
}: {
  device: CaptureDevice;
  now: number;
  attrs: Record<CaptureDevicePresence, BadgeAttrProps>;
}) {
  const t = useT();
  const presence = attrs[captureDevicePresence(device)];

  return (
    <>
      <Badge variant={phaseTone(presence.phase)} title={presence.description}>
        {presence.text}
      </Badge>
      {device.status !== "Revoked" && !device.isOnline && (
        <span className="text-foreground-subtle text-xs">
          {device.lastSeenAt === null
            ? t("Never connected")
            : t("Last seen {0}", formatSecondsAgo(Math.max(0, now - device.lastSeenAt)))}
        </span>
      )}
    </>
  );
}

function DeviceDetail({ device, showOwner }: { device: AnyDevice; showOwner: boolean }) {
  const t = useT();

  return (
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
  );
}

function DeviceReach({ device }: { device: CaptureDevice }) {
  const t = useT();

  if (device.status === "Revoked") {
    return (
      <p className="text-foreground-subtle text-xs">
        {[
          device.revokedAt === null ? null : t("Revoked {0}", formatUnixDateTime(device.revokedAt)),
          device.revokedReason === "" ? null : device.revokedReason,
        ]
          .filter(Boolean)
          .join(" · ")}
      </p>
    );
  }

  return (
    <p className="text-foreground-subtle text-xs">
      {device.sources.length > 0
        ? t("Scanners: {0}", device.sources.map((source) => source.name).join(", "))
        : t("No scanner reported yet")}
    </p>
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
  bordered = true,
}: DeviceListProps) {
  const t = useT();
  const clock = useNowSeconds(PRESENCE_TICK_MS);
  const current = Math.max(now ?? 0, clock);
  const attrs = useMemo(() => captureDevicePresenceAttrs(t), [t]);
  const [revoking, setRevoking] = useState<CaptureDevice | null>(null);

  return (
    <>
      <ul
        className={cn(
          "divide-border-subtle bg-card divide-y",
          bordered && "border-border rounded-lg border",
        )}
      >
        {devices.map((device) => (
          <li key={device.id} className="flex flex-col gap-2 px-3 py-3 sm:flex-row sm:items-start">
            <div className="flex min-w-0 flex-1 flex-col gap-1">
              <div className="flex flex-wrap items-center gap-2">
                <span
                  className={cn(
                    "truncate text-sm",
                    device.status === "Revoked" && "text-foreground-muted",
                  )}
                >
                  {device.name}
                </span>
                <Presence device={device} now={current} attrs={attrs} />
              </div>
              <DeviceDetail device={device} showOwner={showOwner} />
              <DeviceReach device={device} />
            </div>
            {canRevoke && device.status === "Active" && (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setRevoking(device)}
                isLoading={revokingId === device.id}
                loadingText={t("Revoking")}
              >
                <UnplugIcon className="size-3.5" aria-hidden />
                {t("Revoke")}
              </Button>
            )}
          </li>
        ))}
      </ul>

      <RevokeDeviceDialog
        device={revoking}
        onClose={() => setRevoking(null)}
        onConfirm={(reason) => (revoking === null ? undefined : onRevoke(revoking, reason))}
      />
    </>
  );
}
