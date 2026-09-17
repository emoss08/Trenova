import { useT } from "@trenova/shared/i18n/use-t";
import { BrandLogo } from "@trenova/shared/components/brand-logo";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Card } from "@trenova/shared/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import type { AIProviderCatalog } from "@/types/ai-provider";
import {
  KeyRoundIcon,
  PencilIcon,
  PlugZapIcon,
  ServerIcon,
  ShieldCheckIcon,
  Trash2Icon,
} from "lucide-react";
import { useMemo } from "react";
import { providerBrandDomain } from "./provider-brand";
import { ProviderTestSummary } from "./provider-test-summary";

type ProviderCardProps = {
  provider: AIProviderRow;
  catalog: AIProviderCatalog | undefined;
  isTesting: boolean;
  canManage: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  onTest: () => void;
  onEdit: () => void;
  onDelete: () => void;
};

export function ProviderCard({
  provider,
  catalog,
  isTesting,
  canManage,
  canUpdate,
  canDelete,
  onTest,
  onEdit,
  onDelete,
}: ProviderCardProps) {
  const t = useT();

  const domain = useMemo(
    () => providerBrandDomain(provider, catalog?.presets ?? []),
    [provider, catalog?.presets],
  );
  const kindLabel =
    catalog?.kinds.find((kind) => kind.kind === provider.kind)?.label ?? provider.kind;
  const taskLabels = useMemo(() => {
    const byTask = new Map((catalog?.tasks ?? []).map((task) => [task.task, task.label]));
    return provider.tasks.map((task) => byTask.get(task) ?? task);
  }, [provider.tasks, catalog?.tasks]);

  return (
    <Card
      size="sm"
      className={cn(
        "hover:border-primary/40 gap-3 px-3 transition-colors",
        !provider.enabled && "opacity-75",
      )}
    >
      <div className="flex items-start gap-3">
        <BrandLogo domain={domain} name={provider.name} size={40} />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
            <button
              type="button"
              onClick={canUpdate ? onEdit : undefined}
              className={cn("truncate text-sm font-semibold", canUpdate && "hover:underline")}
            >
              {provider.name}
            </button>
            <Badge variant={provider.enabled ? "active" : "inactive"}>
              {provider.enabled ? t("Enabled") : t("Disabled")}
            </Badge>
            {provider.trusted && (
              <Badge variant="info" className="gap-1">
                <ShieldCheckIcon className="size-3" />
                {t("Trusted")}
              </Badge>
            )}
          </div>
          <p className="text-muted-foreground mt-0.5 truncate font-mono text-xs">
            {provider.model}
          </p>
        </div>
        <span className="text-muted-foreground shrink-0 text-[11px] tabular-nums">
          {t("Priority {0}", provider.priority)}
        </span>
      </div>

      <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px]">
        <span>{kindLabel}</span>
        <span aria-hidden>·</span>
        <span className="truncate font-mono">{provider.baseUrl || t("Provider default")}</span>
        {provider.allowPrivateNetwork && (
          <Tooltip>
            <TooltipTrigger
              render={
                <span className="inline-flex items-center gap-1">
                  <ServerIcon className="size-3" />
                  {t("Private network")}
                </span>
              }
            />
            <TooltipContent>{t("May reach a server on your own network")}</TooltipContent>
          </Tooltip>
        )}
        {provider.hasApiKey && (
          <span className="inline-flex items-center gap-1">
            <KeyRoundIcon className="size-3" />
            {t("Key stored")}
          </span>
        )}
      </div>

      {provider.description && (
        <p className="text-muted-foreground line-clamp-2 text-xs">{provider.description}</p>
      )}

      <div className="flex flex-wrap gap-1">
        {taskLabels.length === 0 ? (
          <span className="text-muted-foreground text-[11px]">{t("No tasks assigned")}</span>
        ) : (
          taskLabels.map((label) => (
            <Badge key={label} variant="secondary" className="text-[10px]">
              {label}
            </Badge>
          ))
        )}
      </div>

      <div className="border-border mt-auto flex items-center justify-between gap-2 border-t pt-3">
        <ProviderTestSummary outcome={provider.lastTest} className="min-w-0" />
        <div className="flex shrink-0 items-center gap-1">
          {canManage && (
            <Button size="xs" variant="outline" onClick={onTest} isLoading={isTesting}>
              <PlugZapIcon className="size-3" />
              {t("Test")}
            </Button>
          )}
          {canUpdate && (
            <Button size="icon-xs" variant="ghost" aria-label={t("Edit provider")} onClick={onEdit}>
              <PencilIcon className="size-3.5" />
            </Button>
          )}
          {canDelete && (
            <Button
              size="icon-xs"
              variant="ghost"
              aria-label={t("Remove provider")}
              className="text-muted-foreground hover:text-destructive"
              onClick={onDelete}
            >
              <Trash2Icon className="size-3.5" />
            </Button>
          )}
        </div>
      </div>
    </Card>
  );
}
