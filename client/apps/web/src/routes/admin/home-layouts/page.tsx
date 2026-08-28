import { PageHeader } from "@/components/page-header";
import { usePermission } from "@/hooks/use-permission";
import { useDeleteHomeLayoutPreset, useHomeLayoutPresets } from "@/hooks/use-home-layout";
import type { HomeLayoutPreset } from "@/lib/graphql/home-layout";
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
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  LayoutGridIcon,
  Loader2Icon,
  LockIcon,
  PlusIcon,
  Trash2Icon,
  TrashIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { useRoleOptions } from "./_components/use-role-options";

export function HomeLayoutsPage() {
  const { data: presets, isLoading } = useHomeLayoutPresets();
  const { data: roles } = useRoleOptions();
  const deletePreset = useDeleteHomeLayoutPreset();
  const [confirming, setConfirming] = useState<HomeLayoutPreset | null>(null);

  const { allowed: canCreate } = usePermission(Resource.HomeLayoutPreset, Operation.Create);
  const { allowed: canDelete } = usePermission(Resource.HomeLayoutPreset, Operation.Delete);

  const roleNames = useMemo(
    () => new Map((roles ?? []).map((role) => [role.id, role.name])),
    [roles],
  );

  const remove = async (preset: HomeLayoutPreset) => {
    try {
      await deletePreset.mutateAsync(preset.id);
      toast.success(`Deleted ${preset.name}`);
      setConfirming(null);
    } catch (error) {
      toast.error(graphQLErrorMessage(error, "Could not delete that home screen"));
    }
  };

  return (
    <div className="flex flex-col p-6">
      <PageHeader
        title="Home Screens"
        description="Author a home screen once and assign it to the roles that should land on it."
        className="p-0 py-4"
        actions={
          canCreate ? (
            <Link to="/admin/home-layouts/new">
              <Button size="sm">
                <PlusIcon className="size-4" />
                New home screen
              </Button>
            </Link>
          ) : undefined
        }
      />

      {isLoading ? (
        <div className="flex flex-col gap-2 pt-2">
          {Array.from({ length: 3 }, (_, index) => (
            <Skeleton key={index} className="h-16 rounded-lg" />
          ))}
        </div>
      ) : !presets || presets.length === 0 ? (
        <EmptyState canCreate={canCreate} />
      ) : (
        <div className="flex flex-col gap-2 pt-2">
          {presets.map((preset) => (
            <PresetRow
              key={preset.id}
              preset={preset}
              roleNames={roleNames}
              canDelete={canDelete}
              onDelete={() => setConfirming(preset)}
            />
          ))}
        </div>
      )}

      <AlertDialog open={confirming != null} onOpenChange={(open) => !open && setConfirming(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia className="bg-destructive/10 text-destructive">
              <TrashIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>Delete home screen</AlertDialogTitle>
            <AlertDialogDescription>
              {confirming && (
                <>
                  <strong>{confirming.name}</strong> reaches {confirming.assignedUserCount}{" "}
                  {confirming.assignedUserCount === 1 ? "person" : "people"}. They will fall back to
                  the next home screen that matches them. This cannot be undone.
                </>
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setConfirming(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deletePreset.isPending}
              onClick={() => confirming && void remove(confirming)}
            >
              {deletePreset.isPending && <Loader2Icon className="mr-2 size-4 animate-spin" />}
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function PresetRow({
  preset,
  roleNames,
  canDelete,
  onDelete,
}: {
  preset: HomeLayoutPreset;
  roleNames: Map<string, string>;
  canDelete: boolean;
  onDelete: () => void;
}) {
  const audience = preset.roleIds
    .map((roleId) => roleNames.get(roleId))
    .filter((name): name is string => name != null);

  return (
    <div className="border-border/80 bg-card hover:border-border flex flex-wrap items-center gap-3 rounded-lg border p-3 transition-colors">
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <Link
            to={`/admin/home-layouts/${preset.id}`}
            className="truncate text-sm font-medium hover:underline"
          >
            {preset.name}
          </Link>
          {preset.isOrgDefault && <Badge variant="secondary">Org default</Badge>}
          {preset.locked && (
            <Badge variant="warning" className="gap-1">
              <LockIcon className="size-2.5" />
              Locked
            </Badge>
          )}
        </div>

        {preset.description && (
          <p className="text-2xs text-muted-foreground truncate">{preset.description}</p>
        )}

        <p className="text-2xs text-muted-foreground">
          {preset.widgets.length} {preset.widgets.length === 1 ? "widget" : "widgets"}
          {" · "}
          {audience.length > 0
            ? audience.join(", ")
            : preset.coreResponsibility
              ? `Everyone in ${preset.coreResponsibility}`
              : preset.isOrgDefault
                ? "Everyone without another assignment"
                : "Not assigned to anyone"}
          {" · "}
          {preset.assignedUserCount} {preset.assignedUserCount === 1 ? "person" : "people"}
        </p>
      </div>

      <div className="flex shrink-0 items-center gap-2">
        <Link to={`/admin/home-layouts/${preset.id}`}>
          <Button variant="outline" size="sm">
            Edit
          </Button>
        </Link>
        {canDelete && (
          <Button
            variant="ghost"
            size="icon"
            className="text-destructive"
            aria-label={`Delete ${preset.name}`}
            onClick={onDelete}
          >
            <Trash2Icon className="size-4" />
          </Button>
        )}
      </div>
    </div>
  );
}

function EmptyState({ canCreate }: { canCreate: boolean }) {
  return (
    <div className="border-border mt-2 flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed py-16 text-center">
      <LayoutGridIcon className="text-muted-foreground/40 size-5" />
      <p className="text-sm font-medium">No home screens yet</p>
      <p className="text-muted-foreground max-w-sm text-xs">
        Until you author one, everyone lands on the home screen Trenova ships for their role.
      </p>
      {canCreate && (
        <Link to="/admin/home-layouts/new" className="pt-1">
          <Button size="sm" variant="outline">
            <PlusIcon className="size-4" />
            Create one
          </Button>
        </Link>
      )}
    </div>
  );
}
