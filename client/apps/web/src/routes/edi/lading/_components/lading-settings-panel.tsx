import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { lading } from "@/lib/queries/lading";
import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import type { LadingCheck, LadingSetting } from "@trenova/shared/edi/lading";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { LadingProject, LadingVersion } from "@trenova/shared/types/lading";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { settingDraft, settingUnit, settingValue, type SettingDraft } from "./studio-model";

/**
 * The partner settings the partner file declares, which a planner changes without a new version.
 * They are set on the active version; on any other version the form shows its declarations.
 */
export function LadingSettingsPanel({
  project,
  version,
  check,
}: {
  project: LadingProject;
  version: LadingVersion;
  check: LadingCheck;
}) {
  const t = useT();
  const settings = check.partner?.settings ?? [];
  const isActive = project.activeVersionId === version.id;

  if (settings.length === 0) {
    return (
      <p className="text-muted-foreground p-3 text-sm">
        {t("The partner file declares no settings. Add one with setting name: Type = default.")}
      </p>
    );
  }
  return (
    <SettingsForm
      key={`${project.id}:${project.version}`}
      project={project}
      settings={settings}
      editable={isActive}
    />
  );
}

function SettingsForm({
  project,
  settings,
  editable,
}: {
  project: LadingProject;
  settings: readonly LadingSetting[];
  editable: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.EDI, Operation.Update);
  const [drafts, setDrafts] = useState<Record<string, SettingDraft>>(() =>
    Object.fromEntries(
      settings.map((setting) => [
        setting.name,
        settingDraft(setting, project.settings[setting.name]),
      ]),
    ),
  );

  const values = settings.map((setting) => {
    const unit = settingUnit(setting, project.settings[setting.name]);
    return {
      setting,
      unit,
      value: settingValue(setting, drafts[setting.name] ?? "", unit),
    };
  });
  const invalid = values.filter((entry) => entry.value === undefined);

  const save = useApiMutation({
    mutationFn: () =>
      apiService.ladingService.updateSettings(
        project.id,
        Object.fromEntries(values.map((entry) => [entry.setting.name, entry.value])),
        project.version,
      ),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: lading._def });
    },
  });

  return (
    <form
      className="flex flex-col gap-3 p-3"
      onSubmit={(event) => {
        event.preventDefault();
        if (invalid.length === 0) save.mutate(undefined);
      }}
    >
      {editable ? null : (
        <p className="text-muted-foreground text-xs">
          {t("Settings are changed on the active version. Activate this one to change them here.")}
        </p>
      )}
      <div className="grid gap-3 sm:grid-cols-2">
        {values.map(({ setting, unit, value }) => {
          const label = setting.doc?.trim() || setting.name;
          const draft = drafts[setting.name] ?? "";
          const id = `lading-setting-${setting.name}`;
          return (
            <div key={setting.name} className="flex flex-col gap-1">
              {setting.type === "Bool" ? (
                <label htmlFor={id} className="flex items-center gap-2 text-sm">
                  <Checkbox
                    id={id}
                    checked={draft === true}
                    disabled={!editable}
                    onCheckedChange={(checked) =>
                      setDrafts((current) => ({ ...current, [setting.name]: checked === true }))
                    }
                  />
                  {label}
                </label>
              ) : (
                <>
                  <label htmlFor={id} className="text-sm font-medium">
                    {label}
                  </label>
                  <div className="flex items-center gap-2">
                    <Input
                      id={id}
                      value={String(draft)}
                      placeholder={setting.default ?? ""}
                      readOnly={!editable}
                      aria-invalid={value === undefined ? true : undefined}
                      onChange={(event) =>
                        setDrafts((current) => ({ ...current, [setting.name]: event.target.value }))
                      }
                    />
                    {unit ? <span className="text-muted-foreground text-xs">{unit}</span> : null}
                  </div>
                </>
              )}
              <p
                className={
                  value === undefined ? "text-danger text-xs" : "text-muted-foreground text-xs"
                }
              >
                {value === undefined
                  ? t("Enter a {0}", setting.type)
                  : setting.range
                    ? t(
                        "{0}, from {1} to {2}; blank keeps {3}",
                        setting.type,
                        setting.range[0],
                        setting.range[1],
                        setting.default ?? "",
                      )
                    : t("{0}; blank keeps {1}", setting.type, setting.default ?? "")}
              </p>
            </div>
          );
        })}
      </div>
      {editable ? (
        <div className="flex justify-end">
          <Button
            type="submit"
            size="sm"
            disabled={!canUpdate || invalid.length > 0}
            isLoading={save.isPending}
          >
            {t("Save settings")}
          </Button>
        </div>
      ) : null}
    </form>
  );
}
