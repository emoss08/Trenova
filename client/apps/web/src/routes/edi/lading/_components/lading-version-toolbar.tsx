import { useT } from "@trenova/shared/i18n/use-t";
import { EDITOR_SAVE_EVENT } from "@/components/formula-editor/tariff-commands";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { lading } from "@/lib/queries/lading";
import { apiService } from "@/services/api";
import { LadingVersionStatusBadge } from "@trenova/shared/components/status-badge";
import {
  ArchiveIcon,
  CheckCircleIcon,
  DotsHorizontalIcon,
  PlayIcon,
  Save01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import type { LadingCheck, LadingFile, LadingSample } from "@trenova/shared/edi/lading";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type {
  LadingProject,
  LadingVersion,
  LadingVersionDetail,
} from "@trenova/shared/types/lading";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";

/** How many days a shadow can be asked to run from the toolbar; the server allows 1 to 30. */
const SHADOW_DAYS = [7, 14, 30] as const;

/**
 * The open version and what can be done with it: pick another version, save or discard a
 * draft's changes, certify a draft, activate or shadow a certified version, stop a shadow,
 * archive, or start a new draft from any version. Ctrl+S in the editor (`:w` in Vim) saves the
 * draft, as the Save button does.
 */
export function LadingVersionToolbar({
  project,
  versions,
  version,
  check,
  working,
  dirty,
  onDiscard,
  onPickVersion,
}: {
  project: LadingProject;
  versions: readonly LadingVersion[];
  version: LadingVersion;
  check: LadingCheck;
  working: { files: LadingFile[]; samples: LadingSample[]; notes: string };
  dirty: boolean;
  onDiscard: () => void;
  onPickVersion: (id: string) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.EDI, Operation.Update);
  const { allowed: canCreate } = usePermission(Resource.EDI, Operation.Create);

  const refresh = () => queryClient.invalidateQueries({ queryKey: lading._def });

  const save = useApiMutation({
    mutationFn: () =>
      apiService.ladingService.saveDraft(version.id, {
        files: working.files,
        samples: working.samples,
        notes: working.notes,
        version: version.version,
      }),
    onSuccess: async (saved: LadingVersion) => {
      queryClient.setQueryData<LadingVersionDetail>(lading.version(version.id).queryKey, {
        version: saved,
        samples: working.samples,
      });
      await queryClient.invalidateQueries({ queryKey: lading.versions(project.id).queryKey });
    },
  });
  const certify = useApiMutation({
    mutationFn: () => apiService.ladingService.certify(version.id),
    onSuccess: refresh,
  });
  const activate = useApiMutation({
    mutationFn: () => apiService.ladingService.activate(version.id),
    onSuccess: refresh,
  });
  const archive = useApiMutation({
    mutationFn: () => apiService.ladingService.archive(version.id),
    onSuccess: refresh,
  });
  const startShadow = useApiMutation({
    mutationFn: (days: number) => apiService.ladingService.startShadow(version.id, days),
    onSuccess: refresh,
  });
  const stopShadow = useApiMutation({
    mutationFn: () => apiService.ladingService.stopShadow(version.id),
    onSuccess: refresh,
  });
  const newDraft = useApiMutation({
    mutationFn: () => apiService.ladingService.newDraft(project.id, version.id),
    onSuccess: async (draft: LadingVersion) => {
      await refresh();
      onPickVersion(draft.id);
    },
  });

  const busy =
    save.isPending ||
    certify.isPending ||
    activate.isPending ||
    archive.isPending ||
    startShadow.isPending ||
    stopShadow.isPending ||
    newDraft.isPending;
  const status = version.status;
  const saveFromEditor = useLatestCallback(() => {
    if (status === "Draft" && canUpdate && dirty && !busy) save.mutate(undefined);
  });
  useEffect(() => {
    window.addEventListener(EDITOR_SAVE_EVENT, saveFromEditor);
    return () => window.removeEventListener(EDITOR_SAVE_EVENT, saveFromEditor);
  }, [saveFromEditor]);
  const canArchive = status !== "Active" && status !== "Shadow" && status !== "Archived";
  const canShadow = status === "Certified" && Boolean(project.activeVersionId);

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Select
        value={version.id}
        items={versions.map((item) => ({ value: item.id, label: t("Version {0}", item.number) }))}
        disabled={dirty}
        onValueChange={(next) => {
          if (next) onPickVersion(next);
        }}
      >
        <SelectTrigger className="w-44" aria-label={t("Version")}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {versions.map((item) => (
            <SelectItem key={item.id} value={item.id}>
              <span className="flex items-center gap-2">
                {t("Version {0}", item.number)}
                <LadingVersionStatusBadge status={item.status} />
              </span>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <LadingVersionStatusBadge status={status} />
      <span className="text-muted-foreground text-xs tabular-nums">
        {check.errors > 0
          ? t("{0, plural, one {# error} other {# errors}}", check.errors)
          : check.warnings > 0
            ? t("{0, plural, one {# warning} other {# warnings}}", check.warnings)
            : t("No problems")}
      </span>
      {dirty ? <span className="text-muted-foreground text-xs">{t("Unsaved changes")}</span> : null}

      <div className="ml-auto flex items-center gap-2">
        {status === "Draft" ? (
          <>
            {dirty ? (
              <Button type="button" variant="outline" size="sm" onClick={onDiscard} disabled={busy}>
                {t("Discard")}
              </Button>
            ) : null}
            <Button
              type="button"
              variant={dirty ? "default" : "outline"}
              size="sm"
              disabled={!canUpdate || !dirty || busy}
              isLoading={save.isPending}
              onClick={() => save.mutate(undefined)}
            >
              <Save01Icon className="size-3.5" />
              {t("Save")}
            </Button>
            <Button
              type="button"
              size="sm"
              variant={dirty ? "outline" : "default"}
              disabled={!canUpdate || dirty || check.errors > 0 || busy}
              title={
                dirty
                  ? t("Save the draft before certifying it")
                  : check.errors > 0
                    ? t("Fix the errors before certifying")
                    : undefined
              }
              isLoading={certify.isPending}
              onClick={() => certify.mutate(undefined)}
            >
              <CheckCircleIcon className="size-3.5" />
              {t("Certify")}
            </Button>
          </>
        ) : null}
        {status === "Certified" || status === "Shadow" ? (
          <Button
            type="button"
            size="sm"
            disabled={!canUpdate || busy}
            isLoading={activate.isPending}
            onClick={() => activate.mutate(undefined)}
          >
            <PlayIcon className="size-3.5" />
            {t("Activate")}
          </Button>
        ) : null}
        {status === "Shadow" ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={!canUpdate || busy}
            isLoading={stopShadow.isPending}
            onClick={() => stopShadow.mutate(undefined)}
          >
            {t("Stop shadow")}
          </Button>
        ) : null}
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                type="button"
                size="icon-sm"
                variant="outline"
                aria-label={t("Version actions")}
                disabled={busy}
              >
                <DotsHorizontalIcon className="size-4" />
              </Button>
            }
          />
          <DropdownMenuContent align="end" className="w-72">
            <DropdownMenuItem
              title={t("New draft from this version")}
              description={t("Copies its files and samples into a draft you can change")}
              descriptionClassProps="whitespace-normal"
              disabled={!canCreate || dirty}
              onClick={() => newDraft.mutate(undefined)}
            />
            {canShadow ? (
              <>
                <DropdownMenuSeparator />
                {SHADOW_DAYS.map((days) => (
                  <DropdownMenuItem
                    key={days}
                    title={t("{0, plural, one {Shadow for # day} other {Shadow for # days}}", days)}
                    description={t(
                      "Runs beside the active version on live messages, sends nothing, and records where they differ",
                    )}
                    descriptionClassProps="whitespace-normal"
                    disabled={!canUpdate}
                    onClick={() => startShadow.mutate(days)}
                  />
                ))}
              </>
            ) : null}
            {canArchive ? (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  title={t("Archive this version")}
                  startContent={<ArchiveIcon className="size-3.5" />}
                  color="danger"
                  disabled={!canUpdate || dirty}
                  onClick={() => archive.mutate(undefined)}
                />
              </>
            ) : null}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  );
}
