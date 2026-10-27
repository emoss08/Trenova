import { grantsToPermissions } from "@/lib/role-bundles";
import type { JobTemplate } from "@/lib/role-api";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import type { AddPermission, FieldSensitivity } from "@trenova/shared/types/role";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";
import { PERMISSION_CATALOG_STALE_TIME, permissionCatalog } from "@/lib/queries/permission-catalog";

const EMPTY_TEMPLATE = "empty";

export type TemplateSelection = {
  templateId: string;
  permissions: AddPermission[];
  maxSensitivity?: FieldSensitivity;
};

type RoleTemplateSelectorProps = {
  selectedTemplate: string | null;
  onSelectTemplate: (selection: TemplateSelection) => void;
};

export function RoleTemplateSelector({
  selectedTemplate,
  onSelectTemplate,
}: RoleTemplateSelectorProps) {
  const t = useT();
  const { data, isLoading } = useQuery({
    ...permissionCatalog.bundles(),
    staleTime: PERMISSION_CATALOG_STALE_TIME,
  });
  const templates = useMemo(() => data?.templates ?? [], [data?.templates]);

  const items = useMemo(
    () => [
      { value: EMPTY_TEMPLATE, label: t("Start empty") },
      ...templates.map((template) => ({ value: template.id, label: template.label })),
    ],
    [templates, t],
  );

  const handleChange = useCallback(
    (value: string | null) => {
      if (!value || value === EMPTY_TEMPLATE) {
        onSelectTemplate({ templateId: EMPTY_TEMPLATE, permissions: [] });
        return;
      }
      const template = templates.find((candidate: JobTemplate) => candidate.id === value);
      if (!template) {
        return;
      }
      onSelectTemplate({
        templateId: template.id,
        permissions: grantsToPermissions(template.grants),
        maxSensitivity: template.maxSensitivity,
      });
    },
    [onSelectTemplate, templates],
  );

  return (
    <Select
      value={selectedTemplate ?? null}
      onValueChange={handleChange}
      items={items}
      disabled={isLoading}
    >
      <SelectTrigger className="h-8 w-56 text-sm" aria-label={t("Start from a job")}>
        <SelectValue placeholder={t("Start from a job")} />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={EMPTY_TEMPLATE}>{t("Start empty")}</SelectItem>
        {templates.map((template) => (
          <SelectItem key={template.id} value={template.id}>
            <span className="flex flex-col">
              <span>{template.label}</span>
              <span className="text-muted-foreground text-xs">{template.description}</span>
            </span>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
