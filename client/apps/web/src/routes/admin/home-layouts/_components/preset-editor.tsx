import { PageHeader } from "@/components/page-header";
import {
  useCreateHomeLayoutPreset,
  useHomeLayoutPreview,
  useHomeWidgetCatalog,
  useUpdateHomeLayoutPreset,
} from "@/hooks/use-home-layout";
import type { HomeLayoutPreset, HomeWidget } from "@/lib/graphql/home-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { EyeIcon, EyeOffIcon } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import { HomeCanvas } from "@/routes/home/_components/home-canvas";
import { useRoleOptions } from "./use-role-options";

const CORE_RESPONSIBILITIES = ["Billing", "Operations", "Finance", "Leadership"];
const NO_RESPONSIBILITY = "__none__";
const MAX_PRIORITY = 1000;

export type PresetEditorProps = {
  /** Absent when authoring a new preset. */
  preset?: HomeLayoutPreset;
};

type Draft = {
  name: string;
  description: string;
  widgets: HomeWidget[];
  roleIds: string[];
  coreResponsibility: string;
  isOrgDefault: boolean;
  locked: boolean;
  priority: number;
};

function toDraft(preset: HomeLayoutPreset | undefined): Draft {
  return {
    name: preset?.name ?? "",
    description: preset?.description ?? "",
    widgets: preset?.widgets ?? [],
    roleIds: preset?.roleIds ?? [],
    coreResponsibility: preset?.coreResponsibility ?? "",
    isOrgDefault: preset?.isOrgDefault ?? false,
    locked: preset?.locked ?? false,
    priority: preset?.priority ?? 0,
  };
}

/**
 * Authoring a preset uses the same canvas users get, in edit mode, so an
 * administrator arranges exactly what their people will see rather than a
 * schematic of it. Everything else on the page answers one question: who lands
 * on this, and may they change it.
 */
export function PresetEditor({ preset }: PresetEditorProps) {
  const navigate = useNavigate();
  const [draft, setDraft] = useState<Draft>(() => toDraft(preset));
  const [previewRoleId, setPreviewRoleId] = useState<string | null>(null);

  const catalog = useHomeWidgetCatalog();
  const roles = useRoleOptions();
  const createPreset = useCreateHomeLayoutPreset();
  const updatePreset = useUpdateHomeLayoutPreset();
  const preview = useHomeLayoutPreview(preset?.id ?? null, previewRoleId);

  const saving = createPreset.isPending || updatePreset.isPending;
  const previewing = previewRoleId != null;

  const patch = (next: Partial<Draft>) => setDraft((prev) => ({ ...prev, ...next }));

  const toggleRole = (roleId: string) =>
    patch({
      roleIds: draft.roleIds.includes(roleId)
        ? draft.roleIds.filter((id) => id !== roleId)
        : [...draft.roleIds, roleId],
    });

  const save = async () => {
    if (draft.name.trim() === "") {
      toast.error("Give this home screen a name");
      return;
    }

    const payload = {
      name: draft.name.trim(),
      description: draft.description.trim() || null,
      widgets: draft.widgets.map(toInput),
      roleIds: draft.roleIds,
      coreResponsibility: draft.coreResponsibility || null,
      isOrgDefault: draft.isOrgDefault,
      locked: draft.locked,
      priority: draft.priority,
    };

    try {
      if (preset) {
        await updatePreset.mutateAsync({ ...payload, id: preset.id, version: preset.version });
        toast.success(`Saved ${payload.name}`);
      } else {
        const created = await createPreset.mutateAsync(payload);
        toast.success(`Created ${created.name}`);
        void navigate(`/admin/home-layouts/${created.id}`, { replace: true });
      }
    } catch (error) {
      toast.error(graphQLErrorMessage(error, "Could not save that home screen"));
    }
  };

  return (
    <div className="flex flex-col p-6">
      <PageHeader
        title={preset ? draft.name || "Home screen" : "New home screen"}
        description="Arrange the widgets, then choose who lands on them."
        className="p-0 py-4"
        actions={
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" onClick={() => void navigate("/admin/home-layouts")}>
              Back
            </Button>
            <Button size="sm" onClick={() => void save()} disabled={saving}>
              {preset ? "Save" : "Create"}
            </Button>
          </div>
        }
      />

      <div className="flex flex-col gap-4 pt-2">
        <div className="grid gap-4 lg:grid-cols-3">
          <div className="flex flex-col gap-3 lg:col-span-2">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="preset-name">Name</Label>
              <Input
                id="preset-name"
                value={draft.name}
                placeholder="Dispatch Home"
                onChange={(event) => patch({ name: event.target.value })}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="preset-description">Description</Label>
              <Textarea
                id="preset-description"
                rows={2}
                value={draft.description}
                placeholder="What this home screen is for."
                onChange={(event) => patch({ description: event.target.value })}
              />
            </div>
          </div>

          <div className="flex flex-col gap-3 rounded-lg border border-border p-3">
            <Toggle
              id="preset-org-default"
              label="Organization default"
              hint="Where anyone without a role assignment lands. Only one preset can hold this."
              checked={draft.isOrgDefault}
              onChange={(isOrgDefault) => patch({ isOrgDefault })}
            />
            <Toggle
              id="preset-locked"
              label="Lock this home screen"
              hint="People assigned it cannot rearrange it. Anything they saved earlier is kept and returns if you unlock."
              checked={draft.locked}
              onChange={(locked) => patch({ locked })}
            />
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="preset-priority">Priority</Label>
              <Input
                id="preset-priority"
                type="number"
                min={0}
                max={MAX_PRIORITY}
                value={draft.priority}
                onChange={(event) => {
                  const parsed = Number(event.target.value);
                  patch({
                    priority: Number.isFinite(parsed)
                      ? Math.min(Math.max(parsed, 0), MAX_PRIORITY)
                      : 0,
                  });
                }}
              />
              <p className="text-2xs text-muted-foreground">
                When someone matches more than one home screen, the highest priority wins.
              </p>
            </div>
          </div>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <div className="flex flex-col gap-2 rounded-lg border border-border p-3">
            <Label>Assign to roles</Label>
            {roles.isLoading ? (
              <p className="text-xs text-muted-foreground">Loading roles…</p>
            ) : (
              <div className="grid max-h-48 gap-1 overflow-y-auto sm:grid-cols-2">
                {(roles.data ?? []).map((role) => (
                  <label
                    key={role.id}
                    className="flex items-center gap-2 rounded border border-border px-2 py-1.5 text-xs"
                  >
                    <Checkbox
                      checked={draft.roleIds.includes(role.id)}
                      onCheckedChange={() => toggleRole(role.id)}
                    />
                    <span className="truncate">{role.name}</span>
                  </label>
                ))}
              </div>
            )}
          </div>

          <div className="flex flex-col gap-2 rounded-lg border border-border p-3">
            <Label htmlFor="preset-responsibility">Or assign by job function</Label>
            <Select
              value={draft.coreResponsibility || NO_RESPONSIBILITY}
              onValueChange={(value) =>
                patch({
                  coreResponsibility: !value || value === NO_RESPONSIBILITY ? "" : String(value),
                })
              }
            >
              <SelectTrigger id="preset-responsibility">
                <SelectValue placeholder="No job function" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NO_RESPONSIBILITY}>No job function</SelectItem>
                {CORE_RESPONSIBILITIES.map((responsibility) => (
                  <SelectItem key={responsibility} value={responsibility}>
                    {responsibility}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-2xs text-muted-foreground">
              Reaches every role tagged with that function, including ones added later.
            </p>

            <div className="mt-1 flex flex-col gap-1.5 border-t border-border pt-2">
              <Label htmlFor="preview-role">Preview as</Label>
              <div className="flex items-center gap-2">
                <Select
                  value={previewRoleId ?? NO_RESPONSIBILITY}
                  onValueChange={(value) =>
                    setPreviewRoleId(!value || value === NO_RESPONSIBILITY ? null : String(value))
                  }
                >
                  <SelectTrigger id="preview-role" className="flex-1">
                    <SelectValue placeholder="Nobody" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value={NO_RESPONSIBILITY}>Editing</SelectItem>
                    {(roles.data ?? []).map((role) => (
                      <SelectItem key={role.id} value={role.id}>
                        {role.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {previewing && (
                  <Button variant="ghost" size="sm" onClick={() => setPreviewRoleId(null)}>
                    <EyeOffIcon className="size-3.5" />
                    Stop
                  </Button>
                )}
              </div>
              <p className="text-2xs text-muted-foreground">
                Shows what a member of that role resolves to today. Unsaved edits are not included.
              </p>
            </div>
          </div>
        </div>

        {previewing ? (
          <div className="flex flex-col gap-2">
            <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <EyeIcon className="size-3.5" />
              Previewing — widgets are read-only and drawn against your own permissions.
            </p>
            <HomeCanvas
              widgets={preview.data?.widgets ?? []}
              catalog={catalog.data}
              catalogLoading={catalog.isLoading}
              editing={false}
              onChange={() => undefined}
            />
          </div>
        ) : (
          <HomeCanvas
            widgets={draft.widgets}
            catalog={catalog.data}
            catalogLoading={catalog.isLoading}
            editing
            onChange={(widgets) => patch({ widgets })}
          />
        )}
      </div>
    </div>
  );
}

function Toggle({
  id,
  label,
  hint,
  checked,
  onChange,
}: {
  id: string;
  label: string;
  hint: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-start gap-2">
      <Switch id={id} checked={checked} onCheckedChange={onChange} />
      <div className="flex min-w-0 flex-col gap-0.5">
        <Label htmlFor={id} className="text-xs">
          {label}
        </Label>
        <p className="text-2xs text-muted-foreground">{hint}</p>
      </div>
    </div>
  );
}

function toInput(widget: HomeWidget) {
  return {
    id: widget.id,
    key: widget.key,
    title: widget.title,
    w: widget.w,
    h: widget.h,
    config: {
      metric: widget.config.metric,
      metrics: widget.config.metrics,
      definitionId: widget.config.definitionId,
      cannedKey: widget.config.cannedKey,
      chartId: widget.config.chartId,
      columnId: widget.config.columnId,
      dashboardId: widget.config.dashboardId,
      text: widget.config.text,
      limit: widget.config.limit,
      windowDays: widget.config.windowDays,
    },
  };
}
