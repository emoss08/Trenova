import { ControlledDocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { captureDocumentCategory, type CaptureRecordKind } from "@/lib/capture";
import {
  createCaptureRequest,
  type CaptureDevice,
  type CaptureRequestMode,
} from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

/** The value the scanner picker holds for "whichever scanner the computer defaults to". */
const DEFAULT_SOURCE = "__default__";
/** The value the profile picker holds for "the organization's default profile". */
const DEFAULT_PROFILE = "__default__";

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-foreground-subtle text-xs font-medium">{label}</span>
      {children}
    </div>
  );
}

function DeviceOption({ device }: { device: CaptureDevice }) {
  const t = useT();
  return (
    <span className="flex items-center gap-2">
      <span
        aria-hidden
        className={cn(
          "size-1.5 shrink-0 rounded-full",
          device.isOnline ? "bg-success" : "bg-foreground-subtle",
        )}
      />
      <span className="truncate">{device.name}</span>
      <span className="text-foreground-subtle text-xs">
        {device.isOnline ? t("Online") : t("Offline")}
      </span>
    </span>
  );
}

/**
 * Asks one of the person's own computers to scan into this record, or to catch
 * the next thing they print. A request only ever reaches the person's own
 * computers: starting a scanner somebody else is standing at is not something
 * this can do.
 */
export function CaptureRequestDialog({
  open,
  onOpenChange,
  mode,
  kind,
  recordId,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: CaptureRequestMode;
  kind: CaptureRecordKind;
  recordId: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const devicesQuery = useQuery({ ...queries.capture.myDevices("Active"), enabled: open });
  const profilesQuery = useQuery({
    ...queries.capture.availableProfiles(),
    enabled: open && mode === "Scan",
  });

  const devices = devicesQuery.data ?? [];
  const [deviceId, setDeviceId] = useState<string | null>(null);
  const [sourceName, setSourceName] = useState(DEFAULT_SOURCE);
  const [profileId, setProfileId] = useState(DEFAULT_PROFILE);
  const [documentTypeId, setDocumentTypeId] = useState("");

  // The computer the person last used, or the one that is online, or the first.
  const chosen =
    devices.find((device) => device.id === deviceId) ??
    devices.find((device) => device.isOnline) ??
    devices[0];
  const sources = chosen?.sources ?? [];

  const request = useApiMutation({
    mutationFn: () => {
      if (chosen === undefined) {
        throw new Error(t("Choose a computer"));
      }
      return createCaptureRequest({
        deviceId: chosen.id,
        mode,
        targetType: kind,
        targetId: recordId,
        documentTypeId: documentTypeId === "" ? null : documentTypeId,
        profileId: mode === "Scan" && profileId !== DEFAULT_PROFILE ? profileId : null,
        sourceName: mode === "Scan" && sourceName !== DEFAULT_SOURCE ? sourceName : null,
      });
    },
    onSuccess: async () => {
      toast.success(
        mode === "Scan"
          ? t("Sent to {0}. Put the pages in the scanner.", chosen?.name ?? "")
          : t(
              "Print to Trenova from any program on {0} in the next ten minutes.",
              chosen?.name ?? "",
            ),
      );
      onOpenChange(false);
      await queryClient.invalidateQueries({
        queryKey: queries.capture.requests(kind, recordId).queryKey,
      });
    },
    resourceName: "Capture request",
  });

  const title = mode === "Scan" ? t("Scan into this record") : t("Print into this record");
  const description =
    mode === "Scan"
      ? t(
          "The scan starts on the computer you choose, and its pages are filed onto this record as they arrive. Anything Trenova cannot place waits in Intake.",
        )
      : t(
          "The next document you print to the Trenova printer on the computer you choose is filed here.",
        );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {devicesQuery.isLoading ? (
          <div className="flex flex-col gap-2" aria-busy="true">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : devices.length === 0 ? (
          <Alert variant="info" size="sm">
            <AlertDescription>
              {t(
                "No computer is set up to scan for you yet. Install Trenova Capture, sign in from its tray icon, and approve the code it shows.",
              )}{" "}
              <Link to="/capture/devices" className="ui-focus-ring text-brand hover:underline">
                {t("My scanners")}
              </Link>
            </AlertDescription>
          </Alert>
        ) : (
          <div className="flex flex-col gap-3">
            <Field label={t("Computer")}>
              <Select
                value={chosen?.id ?? null}
                items={devices.map((device) => ({ value: device.id, label: device.name }))}
                onValueChange={(value) => {
                  setDeviceId(typeof value === "string" ? value : null);
                  setSourceName(DEFAULT_SOURCE);
                }}
              >
                <SelectTrigger aria-label={t("Computer")}>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {devices.map((device) => (
                    <SelectItem key={device.id} value={device.id}>
                      <DeviceOption device={device} />
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            {chosen !== undefined && !chosen.isOnline && (
              <Alert variant="warning" size="sm">
                <AlertDescription>
                  {t(
                    "{0} is not connected right now. The request waits a few minutes for it to come online.",
                    chosen.name,
                  )}
                </AlertDescription>
              </Alert>
            )}

            {mode === "Scan" && (
              <>
                <Field label={t("Scanner")}>
                  <Select
                    value={sourceName}
                    items={[
                      { value: DEFAULT_SOURCE, label: t("The computer's default scanner") },
                      ...sources.map((source) => ({ value: source.name, label: source.name })),
                    ]}
                    onValueChange={(value) =>
                      setSourceName(typeof value === "string" ? value : DEFAULT_SOURCE)
                    }
                  >
                    <SelectTrigger aria-label={t("Scanner")}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={DEFAULT_SOURCE}>
                        {t("The computer's default scanner")}
                      </SelectItem>
                      {sources.map((source) => (
                        <SelectItem key={source.name} value={source.name}>
                          {source.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>

                <Field label={t("Scan settings")}>
                  <Select
                    value={profileId}
                    items={[
                      { value: DEFAULT_PROFILE, label: t("Organization default") },
                      ...(profilesQuery.data ?? []).map((profile) => ({
                        value: profile.id,
                        label: profile.name,
                      })),
                    ]}
                    onValueChange={(value) =>
                      setProfileId(typeof value === "string" ? value : DEFAULT_PROFILE)
                    }
                  >
                    <SelectTrigger aria-label={t("Scan settings")}>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value={DEFAULT_PROFILE}>{t("Organization default")}</SelectItem>
                      {(profilesQuery.data ?? []).map((profile) => (
                        <SelectItem key={profile.id} value={profile.id}>
                          {profile.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </Field>
              </>
            )}

            <ControlledDocumentTypeAutocompleteField
              label={t("Document type")}
              placeholder={t("Optional")}
              category={captureDocumentCategory(kind)}
              value={documentTypeId}
              onValueChange={setDocumentTypeId}
            />
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            onClick={() => request.mutate(undefined)}
            disabled={chosen === undefined}
            isLoading={request.isPending}
            loadingText={t("Sending")}
          >
            {mode === "Scan" ? t("Start scan") : t("Wait for my print")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
