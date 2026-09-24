import type { AgentExtensionCatalogItem } from "@/types/agent-extension";
import { BrandLogo } from "@trenova/shared/components/brand-logo";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { ExternalLinkIcon, WrenchIcon } from "lucide-react";
import { dailyUsageShare, extensionState, type ExtensionState } from "./extension-roster";

type ExtensionCardProps = {
  extension: AgentExtensionCatalogItem;
  canUpdate: boolean;
  onOpen: (extension: AgentExtensionCatalogItem) => void;
};

const STATE_BADGE: Record<
  ExtensionState,
  { label: string; variant: "success" | "warning" | "neutral" }
> = {
  on: { label: "On", variant: "success" },
  needsSetup: { label: "Needs setup", variant: "warning" },
  off: { label: "Off", variant: "neutral" },
};

/**
 * One extension in the marketplace: what it gives agents, whether the
 * organization has it on, and — once it is on — how much of the day's
 * request limit it has spent and what it has cost this month.
 */
export function ExtensionCard({ extension, canUpdate, onOpen }: ExtensionCardProps) {
  const t = useT();
  const state = extensionState(extension);
  const badge = STATE_BADGE[state];
  const share = dailyUsageShare(extension);

  return (
    <article className="bg-card flex flex-col gap-4 rounded-lg border p-4">
      <header className="flex items-start gap-3">
        <BrandLogo domain={extension.brandDomain} name={extension.vendor} size={32} />
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="truncate text-sm font-semibold">{extension.name}</h3>
            <Badge variant={badge.variant} appearance={state === "off" ? "outline" : "subtle"}>
              {t(badge.label)}
            </Badge>
          </div>
          <span className="text-muted-foreground text-xs">
            {t("By {0} · {1}", extension.vendor, extension.categoryLabel)}
          </span>
        </div>
      </header>

      <p className="text-muted-foreground text-sm">{extension.summary}</p>

      <div className="flex flex-wrap gap-1.5">
        {extension.tools.map((tool) => (
          <Badge key={tool.name} variant="neutral" appearance="outline" title={tool.description}>
            <WrenchIcon data-icon="inline-start" aria-hidden />
            <span className="font-mono">{tool.name}</span>
          </Badge>
        ))}
      </div>

      <footer className="mt-auto flex items-center justify-between gap-2">
        <a
          href={extension.websiteUrl}
          target="_blank"
          rel="noreferrer noopener"
          className="ui-focus-ring text-muted-foreground hover:text-foreground inline-flex items-center gap-1 rounded-sm text-xs"
        >
          {extension.vendor}
          <ExternalLinkIcon className="size-3" aria-hidden />
        </a>
        <Button
          size="sm"
          variant={state === "off" ? "default" : "outline"}
          onClick={() => onOpen(extension)}
        >
          {!canUpdate ? t("View") : state === "off" ? t("Set up") : t("Manage")}
        </Button>
      </footer>
    </article>
  );
}
