import { useT } from "@trenova/shared/i18n/use-t";
import { CredentialHealthBadge } from "@trenova/shared/components/credential-health-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { credentialHealthMeta, describeDaysUntil } from "@trenova/shared/lib/credential";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  fetchMyCredentials,
  type PortalCredential,
} from "@trenova/shared/lib/graphql/driver-portal";
import { uploadMyCredentialDocument } from "@trenova/shared/lib/portal";
import { cn } from "@trenova/shared/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CameraIcon, IdCardIcon, ShieldCheckIcon } from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { useDashFeatures } from "./use-dash-features";

export const DASH_CREDENTIALS_KEY = "dash-credentials";

function sortWorstFirst(items: readonly PortalCredential[]): PortalCredential[] {
  return [...items].sort((a, b) => {
    const rank = credentialHealthMeta(a.health).rank - credentialHealthMeta(b.health).rank;
    if (rank !== 0) return rank;
    if (a.required !== b.required) return a.required ? -1 : 1;
    return a.name.localeCompare(b.name);
  });
}

function canUploadRenewal(item: PortalCredential): boolean {
  return Boolean(item.id) && (item.health === "ExpiringSoon" || item.health === "Expired");
}

export function CredentialsCard() {
  const t = useT();

  const features = useDashFeatures();
  const credentials = useQuery({
    queryKey: [DASH_CREDENTIALS_KEY],
    queryFn: ({ signal }) => fetchMyCredentials({ signal }),
  });

  const items = useMemo(() => sortWorstFirst(credentials.data ?? []), [credentials.data]);
  const attention = items.filter((item) => item.health !== "Valid").length;

  if (credentials.isPending) {
    return <Skeleton className="h-40 w-full rounded-2xl" />;
  }
  if (!credentials.data) {
    return null;
  }

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <IdCardIcon className="size-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold">{t("Credentials")}</h2>
        </div>
        {attention > 0 ? (
          <Badge variant="warning">
            {t("{0} need{1} attention", attention, attention === 1 ? "s" : "")}
          </Badge>
        ) : (
          <Badge variant="active">{t("All current")}</Badge>
        )}
      </div>

      {items.length === 0 ? (
        <p className="mt-3 text-xs text-muted-foreground">
          {t("Your carrier has not listed any credentials for you yet.")}
        </p>
      ) : (
        <ul className="mt-3 divide-y divide-border border-t border-border">
          {items.map((item) => (
            <CredentialRow
              key={item.credentialTypeId}
              item={item}
              canUpload={features.allowProfileDocumentUpload && canUploadRenewal(item)}
            />
          ))}
        </ul>
      )}
      <p className="mt-3 text-xs text-muted-foreground">
        {t("Renewed a card? Upload a photo and your carrier will update the record after checking it.")}
      </p>
    </div>
  );
}

function CredentialRow({ item, canUpload }: { item: PortalCredential; canUpload: boolean }) {
  const t = useT();

  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [fileName, setFileName] = useState<string | null>(null);
  const meta = credentialHealthMeta(item.health);

  const upload = useMutation({
    mutationFn: (file: File) => {
      if (!item.id) throw new Error("Nothing to renew yet");
      return uploadMyCredentialDocument(item.id, file);
    },
    onSuccess: async () => {
      toast.success(t("Uploaded — your carrier will review it and update the record."));
      setFileName(null);
      await queryClient.invalidateQueries({ queryKey: [DASH_CREDENTIALS_KEY] });
    },
    onError: (error: Error) => {
      setFileName(null);
      toast.error(error.message || "Upload failed. Try again.");
    },
  });

  return (
    <li
      data-testid={`dash-credential-${item.credentialTypeId}`}
      className="flex flex-col gap-1.5 py-2.5"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">
            {item.name}
            {item.required ? (
              <span className="ml-1 text-[10px] font-normal uppercase text-muted-foreground">
                {t("Required")}
              </span>
            ) : null}
          </p>
          <p className={cn("text-xs", meta.textClass)}>
            {item.health === "Missing" || item.expiresAt ? (
              <span>{describeDaysUntilOrMissing(item)}</span>
            ) : (
              <span>{t("No expiry")}</span>
            )}
            {item.expiresAt && item.health !== "Missing" ? (
              <span className="text-muted-foreground"> · {formatUnixDate(item.expiresAt)}</span>
            ) : null}
          </p>
        </div>
        <CredentialHealthBadge health={item.health} daysUntilExpiry={item.daysUntilExpiry} />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
        <div className="flex items-center gap-2 text-muted-foreground">
          {item.numberMasked ? <span className="tabular-nums">{item.numberMasked}</span> : null}
          {item.verified ? (
            <span className="flex items-center gap-1 text-green-600 dark:text-green-400">
              <ShieldCheckIcon className="size-3.5" />
              {t("Verified")}
            </span>
          ) : null}
        </div>
        {canUpload ? (
          <>
            <input
              ref={fileInputRef}
              data-testid={`dash-credential-file-${item.credentialTypeId}`}
              type="file"
              accept="image/*,application/pdf"
              capture="environment"
              className="hidden"
              onChange={(event) => {
                const file = event.target.files?.[0];
                event.target.value = "";
                if (file) {
                  setFileName(file.name);
                  upload.mutate(file);
                }
              }}
            />
            <Button
              variant="outline"
              size="sm"
              className="h-8"
              aria-label={`Upload renewed ${item.name}`}
              disabled={upload.isPending}
              onClick={() => fileInputRef.current?.click()}
            >
              <CameraIcon className="size-3.5" />
              {upload.isPending ? t("Uploading {0}…", fileName ?? "") : t("Upload renewal")}
            </Button>
          </>
        ) : null}
      </div>
    </li>
  );
}

function describeDaysUntilOrMissing(item: PortalCredential): string {
  if (item.health === "Missing") return "Not on file — ask your fleet manager";
  return describeDaysUntil(item.daysUntilExpiry);
}
