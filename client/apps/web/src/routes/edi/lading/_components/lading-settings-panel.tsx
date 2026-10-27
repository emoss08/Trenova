import { useT } from "@trenova/shared/i18n/use-t";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { lading } from "@/lib/queries/lading";
import { apiService } from "@/services/api";
import { Lock01Icon, ReverseLeftIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Switch } from "@trenova/shared/components/ui/switch";
import type { LadingCheck, LadingSetting } from "@trenova/shared/edi/lading";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { LadingProject, LadingVersion } from "@trenova/shared/types/lading";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { cn } from "@trenova/shared/lib/utils";
import {
  PanelSheet,
} from "./lading-panel-kit";
import { SettingsSketch } from "./lading-panel-sketches";
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
      <PanelSheet
        sketch={<SettingsSketch />}
        title={t("No partner settings")}
        description={t(
          "The partner file declares no settings. Add one with setting name: Type = default.",
        )}
      />
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
  const [saved] = useState<Record<string, SettingDraft>>(() =>
    Object.fromEntries(
      settings.map((setting) => [
        setting.name,
        settingDraft(setting, project.settings[setting.name]),
      ]),
    ),
  );
  const [drafts, setDrafts] = useState(saved);

  const values = settings.map((setting) => {
    const unit = settingUnit(setting, project.settings[setting.name]);
    return {
      setting,
      unit,
      value: settingValue(setting, drafts[setting.name] ?? "", unit),
      changed: drafts[setting.name] !== saved[setting.name],
    };
  });
  const invalid = values.filter((entry) => entry.value === undefined);
  const changed = values.filter((entry) => entry.changed).length;

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

  const blank = (setting: LadingSetting): SettingDraft =>
    setting.type === "Bool" ? setting.default?.trim() === "true" : "";

  return (
    <form
      className="flex min-h-full flex-col"
      onSubmit={(event) => {
        event.preventDefault();
        if (invalid.length === 0) save.mutate(undefined);
      }}
    >
      {editable ? null : (
        <Alert variant="info" size="sm" className="rounded-none border-x-0 border-t-0">
          <Lock01Icon />
          <AlertDescription>
            {t("Settings are changed on the active version. Activate this one to change them here.")}
          </AlertDescription>
        </Alert>
      )}
      <div role="table" aria-label={t("Partner settings")} className="flex-1 text-sm">
        <div
          role="row"
          className="bg-muted text-muted-foreground hidden grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,1fr)_2rem] gap-4 border-b px-3 py-1.5 text-xs font-medium md:grid"
        >
          <span role="columnheader">{t("Setting")}</span>
          <span role="columnheader">{t("Value")}</span>
          <span role="columnheader">{t("Default")}</span>
          <span role="columnheader" className="sr-only">
            {t("Reset")}
          </span>
        </div>
        {values.map(({ setting, unit, value, changed: rowChanged }) => {
          const label = setting.doc?.trim() || setting.name;
          const draft = drafts[setting.name] ?? "";
          const id = `lading-setting-${setting.name}`;
          const set = (next: SettingDraft) =>
            setDrafts((current) => ({ ...current, [setting.name]: next }));
          return (
            <div
              key={setting.name}
              role="row"
              className="grid grid-cols-[minmax(0,1fr)_2rem] items-start gap-x-4 gap-y-2 border-b px-3 py-2.5 md:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_minmax(0,1fr)_2rem]"
            >
              <div role="cell" className="flex min-w-0 flex-col gap-0.5">
                <label htmlFor={id} className="flex items-center gap-1.5">
                  {rowChanged ? (
                    <span
                      aria-hidden
                      title={t("Changed")}
                      className="bg-brand size-1.5 shrink-0 rounded-full"
                    />
                  ) : null}
                  <span className="truncate">{label}</span>
                </label>
                <span className="text-muted-foreground flex items-center gap-1.5 text-xs">
                  <span className="truncate font-mono">{setting.name}</span>
                  <Badge variant="neutral" appearance="outline" className="font-mono">
                    {setting.type}
                  </Badge>
                </span>
              </div>
              <div role="cell" className="col-start-1 flex min-w-0 flex-col gap-1 md:col-start-auto">
                {setting.type === "Bool" ? (
                  <span className="flex h-7 items-center gap-2">
                    <Switch
                      id={id}
                      checked={draft === true}
                      disabled={!editable}
                      onCheckedChange={(checked) => set(checked === true)}
                    />
                    <span className="text-muted-foreground text-xs">
                      {draft === true ? t("On") : t("Off")}
                    </span>
                  </span>
                ) : (
                  <Input
                    id={id}
                    value={String(draft)}
                    placeholder={setting.default ?? ""}
                    readOnly={!editable}
                    sideText={unit || undefined}
                    className="font-mono"
                    aria-invalid={value === undefined ? true : undefined}
                    aria-describedby={value === undefined ? `${id}-error` : undefined}
                    onChange={(event) => set(event.target.value)}
                  />
                )}
                {value === undefined ? (
                  <p id={`${id}-error`} className="text-danger text-xs">
                    {t("Enter a {0}", setting.type)}
                  </p>
                ) : null}
              </div>
              <div
                role="cell"
                className="text-muted-foreground col-start-1 flex min-w-0 flex-col gap-0.5 text-xs md:col-start-auto"
              >
                <span className="truncate font-mono" title={setting.default ?? undefined}>
                  {setting.default ?? "—"}
                </span>
                {setting.range ? (
                  <span className="truncate">
                    {t("From {0} to {1}", setting.range[0], setting.range[1])}
                  </span>
                ) : null}
              </div>
              <div role="cell" className="col-start-2 row-start-1 md:col-start-auto md:row-start-auto">
                {editable && draft !== blank(setting) ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    aria-label={t("Reset {0} to its default", setting.name)}
                    title={t("Reset to the default")}
                    onClick={() => set(blank(setting))}
                  >
                    <ReverseLeftIcon className="size-3.5" />
                  </Button>
                ) : null}
              </div>
            </div>
          );
        })}
      </div>
      {editable ? (
        <div className="bg-background sticky bottom-0 flex items-center gap-2 border-t px-3 py-2">
          <span
            className={cn(
              "flex-1 text-xs",
              invalid.length > 0 ? "text-danger" : "text-muted-foreground",
            )}
          >
            {invalid.length > 0
              ? t(
                  "{0, plural, one {# setting needs a valid value} other {# settings need a valid value}}",
                  invalid.length,
                )
              : changed > 0
                ? t("{0, plural, one {# unsaved change} other {# unsaved changes}}", changed)
                : t("A blank value keeps the default")}
          </span>
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={changed === 0}
            onClick={() => setDrafts(saved)}
          >
            {t("Discard")}
          </Button>
          <Button
            type="submit"
            size="sm"
            disabled={!canUpdate || invalid.length > 0 || changed === 0}
            isLoading={save.isPending}
          >
            {t("Save settings")}
          </Button>
        </div>
      ) : null}
    </form>
  );
}
