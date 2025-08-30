/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { LazyImage } from "@/components/ui/image";
import { ExternalLink } from "@/components/ui/link";
import type { IntegrationSchema } from "@/lib/schemas/integration-schema";
import { CheckCircle2 } from "lucide-react";
import { integrationImages } from "../_utils/integration";

export default function IntegrationCard({
  integration,
  handleConfigureClick,
}: {
  integration: IntegrationSchema;
  handleConfigureClick: (integration: IntegrationSchema) => void;
}) {
  const getButtonText = (integration: IntegrationSchema) => {
    return integration.enabled ? "Manage" : "Enable";
  };

  return (
    <div className="group relative overflow-hidden rounded-[0.65rem] border border-input bg-card p-4 transition-all">
      <div className="pointer-events-none absolute -right-6 -top-8 h-20 w-20 rounded-full bg-[radial-gradient(circle_at_70%_30%,_rgba(99,102,241,0.25),_transparent_60%)] sm:h-28 sm:w-28 sm:-right-8 sm:-top-10" />
      <div className="flex flex-row items-center justify-between gap-4">
        <div className="flex flex-col min-w-0">
          <div className="flex items-center gap-2">
            <div className="text-base sm:text-lg md:text-xl font-semibold tracking-tight truncate">
              {integration.name}
            </div>
            {integration.featured && (
              <Badge
                variant="warning"
                withDot={false}
                className="h-5 px-1.5 shrink-0"
              >
                Featured
              </Badge>
            )}
          </div>
          <div className="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            <Badge
              variant="secondary"
              withDot={false}
              className="h-5 px-1.5 shrink-0"
            >
              By {integration.builtBy}
            </Badge>
            {integration.enabled && (
              <span className="inline-flex items-center gap-1 text-emerald-600">
                <CheckCircle2 className="size-3.5" />
                Enabled
              </span>
            )}
          </div>
        </div>
        <div className="relative flex-shrink-0">
          <div className="absolute inset-0 -m-1 rounded-full bg-[radial-gradient(circle_at_50%_50%,_rgba(56,189,248,0.35),_transparent_60%)] blur-md" />
          <div className="relative flex items-center justify-center rounded-full border border-input bg-background p-2">
            <LazyImage
              src={
                (integration.logoUrl ||
                  integrationImages[integration.type]) as string
              }
              className="size-6"
            />
          </div>
        </div>
      </div>
      <div className="mt-2">
        <div className="min-h-[72px] md:min-h-[80px] text-wrap truncate text-sm text-muted-foreground">
          {integration.description}
        </div>
      </div>

      <div className="mt-2 border-t border-dashed border-input pt-3 flex flex-wrap items-center gap-2">
        <Button
          onClick={(e) => {
            e.stopPropagation();
            handleConfigureClick(integration);
          }}
          type="button"
          variant={integration.enabled ? "outline" : "default"}
          className="w-full sm:w-auto"
        >
          {getButtonText(integration)}
        </Button>
        <div className="flex items-center gap-1 ml-auto">
          {integration.docsUrl && (
            <>
              <ExternalLink
                href={integration.docsUrl}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Docs
              </ExternalLink>
            </>
          )}
          {integration.websiteUrl && (
            <>
              <span aria-hidden className="opacity-50">
                •
              </span>
              <ExternalLink
                href={integration.websiteUrl}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Website
              </ExternalLink>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
