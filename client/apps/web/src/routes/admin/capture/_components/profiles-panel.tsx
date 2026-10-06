import { SectionPanel } from "@/components/section-panel";
import { usePermission, usePermissions } from "@/hooks/use-permission";
import { capturePixelTypeLabel, captureSeparatorLabel } from "@/lib/capture";
import { deleteCaptureProfile, type CaptureProfile } from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
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
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Edit02Icon, PlusIcon, Trash01Icon } from "@trenova/shared/components/icons";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { ProfilePanel, type ProfilePanelMode } from "./profile-panel";

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

function ProfileRow({
  profile,
  canUpdate,
  canDelete,
  onEdit,
  onDelete,
}: {
  profile: CaptureProfile;
  canUpdate: boolean;
  canDelete: boolean;
  onEdit: (profile: CaptureProfile) => void;
  onDelete: (profile: CaptureProfile) => void;
}) {
  const t = useT();

  return (
    <li className="flex items-start gap-3 px-3 py-3">
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium">{profile.name}</span>
          {profile.isDefault && (
            <Badge variant="neutral" appearance="outline">
              {t("Default")}
            </Badge>
          )}
          {profile.status === "Inactive" && <Badge variant="neutral">{t("Not offered")}</Badge>}
        </div>
        <ProfileSummary profile={profile} />
        {profile.description !== "" && (
          <p className="text-foreground-subtle text-xs">{profile.description}</p>
        )}
      </div>
      {(canUpdate || canDelete) && (
        <div className="flex shrink-0 items-center gap-1">
          {canUpdate && (
            <Button
              size="icon-sm"
              variant="ghost"
              aria-label={t("Edit {0}", profile.name)}
              onClick={() => onEdit(profile)}
            >
              <Edit02Icon className="size-3.5" />
            </Button>
          )}
          {canDelete && (
            <Button
              size="icon-sm"
              variant="ghost"
              aria-label={t("Delete {0}", profile.name)}
              onClick={() => onDelete(profile)}
            >
              <Trash01Icon className="size-3.5" />
            </Button>
          )}
        </div>
      )}
    </li>
  );
}

/** The organization's scanning profiles, for the people who manage them. */
export function ProfilesPanel() {
  const t = useT();
  const queryClient = useQueryClient();
  const { canCreate, canUpdate } = usePermissions(Resource.CaptureProfile);
  const { allowed: canDelete } = usePermission(Resource.CaptureProfile, Operation.Delete);
  const profilesQuery = useQuery(queries.capture.profiles(null, ""));

  const [panelMode, setPanelMode] = useState<ProfilePanelMode>("create");
  const [panelOpen, setPanelOpen] = useState(false);
  const [editing, setEditing] = useState<CaptureProfile | null>(null);
  // The profile stays set while the confirmation closes, so its name does not
  // drop out of the title mid-animation.
  const [deleting, setDeleting] = useState<CaptureProfile | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);

  const refresh = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.capture.profiles._def }),
      queryClient.invalidateQueries({ queryKey: queries.capture.availableProfiles._def }),
    ]);
  }, [queryClient]);

  const remove = useMutation({
    mutationFn: (profile: CaptureProfile) => deleteCaptureProfile(profile.id),
    onSuccess: async () => {
      toast.success(t("Profile deleted"));
      setDeleteOpen(false);
      await refresh();
    },
  });

  const handleSaved = useCallback(
    async (saved: CaptureProfile) => {
      setEditing((current) => (current?.id === saved.id ? saved : current));
      await refresh();
    },
    [refresh],
  );

  const openCreate = () => {
    setPanelMode("create");
    setPanelOpen(true);
  };

  const openEdit = (profile: CaptureProfile) => {
    setEditing(profile);
    setPanelMode("edit");
    setPanelOpen(true);
  };

  const askDelete = (profile: CaptureProfile) => {
    remove.reset();
    setDeleting(profile);
    setDeleteOpen(true);
  };

  const handleDeleteOpenChange = (open: boolean) => {
    if (!open && remove.isPending) {
      return;
    }
    setDeleteOpen(open);
  };

  const profiles = profilesQuery.data ?? [];

  return (
    <>
      <SectionPanel
        title={t("Scan profiles")}
        count={profiles.length}
        help={t(
          "What people pick from when they start a scan. The default is used when nobody picks one.",
        )}
        action={
          canCreate ? (
            <Button size="sm" onClick={openCreate}>
              <PlusIcon className="size-3.5" />
              {t("New scan profile")}
            </Button>
          ) : undefined
        }
      >
        {profilesQuery.isPending ? (
          <div className="flex flex-col gap-2 p-3" aria-busy="true">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : profilesQuery.isError ? (
          <ErrorState
            error={profilesQuery.error}
            title={t("Scan profiles could not be loaded.")}
            onRetry={() => void profilesQuery.refetch()}
          />
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
          <ul className="divide-border-subtle divide-y">
            {profiles.map((profile) => (
              <ProfileRow
                key={profile.id}
                profile={profile}
                canUpdate={canUpdate}
                canDelete={canDelete}
                onEdit={openEdit}
                onDelete={askDelete}
              />
            ))}
          </ul>
        )}
      </SectionPanel>

      <ProfilePanel
        mode={panelMode}
        open={panelOpen}
        onOpenChange={setPanelOpen}
        profile={editing}
        onSaved={handleSaved}
      />

      <AlertDialog open={deleteOpen} onOpenChange={handleDeleteOpenChange}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia className="bg-danger-subtle text-destructive">
              <Trash01Icon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Delete {0}?", deleting?.name ?? "")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "It stops being offered. A scan waiting to start with it uses the default instead, and stacks already scanned keep the settings they were scanned with.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {remove.isError && (
            <Alert variant="destructive" size="sm">
              <AlertDescription>
                {graphQLErrorMessage(
                  remove.error,
                  t("Something went wrong. Try again in a moment."),
                )}
              </AlertDescription>
            </Alert>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={remove.isPending}>{t("Keep it")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              isLoading={remove.isPending}
              onClick={() => {
                if (deleting !== null) {
                  remove.mutate(deleting);
                }
              }}
            >
              {t("Delete profile")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
