import { useT } from "@trenova/shared/i18n/use-t";
import { BrandLogo } from "@trenova/shared/components/brand-logo";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
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

export type ProviderRowActions = {
  canManage: boolean;
  canUpdate: boolean;
  canDelete: boolean;
  isTesting: (provider: AIProviderRow) => boolean;
  onTest: (provider: AIProviderRow) => void;
  onEdit: (provider: AIProviderRow) => void;
  onDelete: (provider: AIProviderRow) => void;
};

/**
 * Providers as the list work is offered to: one ruled row each, in routing
 * order, with the priority number on the left so the order can be read off
 * the page. A grid of cards had the same facts in it and no order at all.
 */
export function ProviderRows({
  providers,
  catalog,
  actions,
}: {
  providers: AIProviderRow[];
  catalog: AIProviderCatalog | undefined;
  actions: ProviderRowActions;
}) {
  return (
    <ol className="bg-card divide-border divide-y overflow-hidden rounded-lg border">
      {providers.map((provider, index) => (
        <ProviderRow
          key={provider.id}
          provider={provider}
          position={index + 1}
          catalog={catalog}
          actions={actions}
        />
      ))}
    </ol>
  );
}

export function ProviderRow({
  provider,
  position,
  catalog,
  actions,
}: {
  provider: AIProviderRow;
  position: number;
  catalog: AIProviderCatalog | undefined;
  actions: ProviderRowActions;
}) {
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
    <li
      className={cn(
        "hover:bg-surface-hover grid grid-cols-[auto_auto_minmax(0,1fr)_auto] items-center gap-x-3 px-3 py-2.5 transition-colors",
        !provider.enabled && "text-muted-foreground",
      )}
    >
      <span
        className="text-muted-foreground w-5 text-right text-xs tabular-nums"
        title={t("Priority {0}", provider.priority)}
      >
        {position}
      </span>
      <BrandLogo
        domain={domain}
        name={provider.name}
        size={28}
        className={cn(!provider.enabled && "opacity-60 grayscale")}
      />

      <div className="flex min-w-0 flex-col gap-0.5">
        <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
          <button
            type="button"
            onClick={actions.canUpdate ? () => actions.onEdit(provider) : undefined}
            className={cn(
              "ui-focus-ring truncate rounded-control text-sm font-medium",
              actions.canUpdate && "hover:underline",
              provider.enabled ? "text-foreground" : "text-muted-foreground",
            )}
          >
            {provider.name}
          </button>
          <span className="text-muted-foreground truncate font-mono text-xs">{provider.model}</span>
          {!provider.enabled && (
            <Badge variant="neutral" appearance="outline">
              {t("Off")}
            </Badge>
          )}
          {provider.trusted && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <Badge variant="info" className="gap-1">
                    <ShieldCheckIcon className="size-3" />
                    {t("Trusted")}
                  </Badge>
                }
              />
              <TooltipContent>{t("May take tasks that read sensitive records")}</TooltipContent>
            </Tooltip>
          )}
        </div>
        <div className="text-muted-foreground flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 text-xs">
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
          <span aria-hidden>·</span>
          {taskLabels.length === 0 ? (
            <span>{t("No tasks assigned")}</span>
          ) : (
            <span className="flex flex-wrap gap-1">
              {taskLabels.map((label) => (
                <Badge key={label} variant="neutral" appearance="outline" className="text-2xs">
                  {label}
                </Badge>
              ))}
            </span>
          )}
        </div>
        <ProviderTestSummary outcome={provider.lastTest} className="min-w-0" />
      </div>

      <div className="flex shrink-0 items-center gap-1">
        {actions.canManage && (
          <Button
            size="xs"
            variant="outline"
            onClick={() => actions.onTest(provider)}
            isLoading={actions.isTesting(provider)}
          >
            <PlugZapIcon className="size-3" />
            {t("Test")}
          </Button>
        )}
        {actions.canUpdate && (
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={t("Edit provider")}
            onClick={() => actions.onEdit(provider)}
          >
            <PencilIcon className="size-3.5" />
          </Button>
        )}
        {actions.canDelete && (
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={t("Remove provider")}
            className="text-muted-foreground hover:text-destructive"
            onClick={() => actions.onDelete(provider)}
          >
            <Trash2Icon className="size-3.5" />
          </Button>
        )}
      </div>
    </li>
  );
}
