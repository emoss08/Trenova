import { usePermission, usePermissions } from "@/hooks/use-permission";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { capturePixelTypeLabel, captureSeparatorLabel } from "@/lib/capture";
import { deleteCaptureProfile, type CaptureProfile } from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
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
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PencilIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { ProfileFormDialog } from "./profile-form-dialog";

function ProfileSummary({ profile }: { profile: CaptureProfile }) {
  const t = useT();
  const parts = [
    t("{0} DPI", profile.dpi),
    capturePixelTypeLabel(t, profile.pixelType),
    profile.duplex ? t("Both sides") : t("One side"),
    profile.separatorStrategies.length === 0
      ? t("No splitting")
      : t(
          "Splits on {0}",
          profile.separatorStrategies
            .map((strategy) =>
              strategy === "FixedPageCount"
                ? t("every {0} pages", profile.fixedPageCount)
                : captureSeparatorLabel(t, strategy).toLowerCase(),
            )
            .join(", "),
        ),
  ];

  return <p className="text-foreground-muted text-xs">{parts.join(" · ")}</p>;
}

/** The organization's scanning profiles, for the people who manage them. */
export function ProfilesPanel() {
  const t = useT();
  const queryClient = useQueryClient();
  const { canCreate, canUpdate } = usePermissions(Resource.CaptureProfile);
  const { allowed: canDelete } = usePermission(Resource.CaptureProfile, Operation.Delete);
  const profilesQuery = useQuery(queries.capture.profiles(null, ""));
  const [editing, setEditing] = useState<CaptureProfile | null>(null);
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<CaptureProfile | null>(null);

  const refresh = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.capture.profiles._def }),
      queryClient.invalidateQueries({ queryKey: queries.capture.availableProfiles._def }),
    ]);
  };

  const remove = useApiMutation({
    mutationFn: (profile: CaptureProfile) => deleteCaptureProfile(profile.id),
    onSuccess: async () => {
      toast.success(t("Profile deleted"));
      await refresh();
    },
    resourceName: "Scan profile",
  });

  const profiles = profilesQuery.data ?? [];

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-2">
        <p className="text-foreground-muted max-w-prose text-sm">
          {t(
            "What people pick from when they start a scan. The default is used when nobody picks one.",
          )}
        </p>
        {canCreate && (
          <Button size="sm" onClick={() => setCreating(true)}>
            <PlusIcon className="size-3.5" />
            {t("New scan profile")}
          </Button>
        )}
      </div>

      {profilesQuery.isLoading ? (
        <div className="flex flex-col gap-2" aria-busy="true">
          <Skeleton className="h-14 w-full" />
          <Skeleton className="h-14 w-full" />
        </div>
      ) : profilesQuery.isError ? (
        <Alert variant="destructive" size="sm">
          <AlertDescription>{t("Scan profiles could not be loaded.")}</AlertDescription>
        </Alert>
      ) : profiles.length === 0 ? (
        <EmptySheet
          title={t("No scan profiles yet")}
          description={t(
            "Without one, each computer scans with its scanner's own settings. Add a profile to set the resolution, color and splitting once for everybody.",
          )}
          sketch={
            <div className="flex flex-col gap-2 px-6">
              <GhostLine className="w-1/3" />
              <GhostLine className="w-2/3" />
            </div>
          }
        />
      ) : (
        <ul className="border-border divide-border-subtle bg-card divide-y rounded-lg border">
          {profiles.map((profile) => (
            <li key={profile.id} className="flex items-start gap-3 px-3 py-3">
              <div className="flex min-w-0 flex-1 flex-col gap-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-sm font-medium">{profile.name}</span>
                  {profile.isDefault && <Badge variant="brand">{t("Default")}</Badge>}
                  {profile.status === "Inactive" && (
                    <Badge variant="neutral">{t("Not offered")}</Badge>
                  )}
                </div>
                <ProfileSummary profile={profile} />
                {profile.description !== "" && (
                  <p className="text-foreground-subtle text-xs">{profile.description}</p>
                )}
              </div>
              {canUpdate && (
                <Button
                  size="icon-sm"
                  variant="ghost"
                  aria-label={t("Edit {0}", profile.name)}
                  onClick={() => setEditing(profile)}
                >
                  <PencilIcon className="size-3.5" />
                </Button>
              )}
              {canDelete && (
                <Button
                  size="icon-sm"
                  variant="ghost"
                  aria-label={t("Delete {0}", profile.name)}
                  onClick={() => setDeleting(profile)}
                  isLoading={remove.isPending && remove.variables?.id === profile.id}
                >
                  <Trash2Icon className="size-3.5" />
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}

      <ProfileFormDialog
        open={creating || editing !== null}
        profile={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={refresh}
      />

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia className="bg-danger-subtle text-destructive">
              <Trash2Icon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Delete {0}?", deleting?.name ?? "")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "It stops being offered. A scan waiting to start with it uses the default instead, and stacks already scanned keep the settings they were scanned with.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Keep it")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => {
                if (deleting !== null) {
                  remove.mutate(deleting);
                }
                setDeleting(null);
              }}
            >
              {t("Delete profile")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
