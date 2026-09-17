import { StatusDot, type StatusTone } from "@/components/carrier-intelligence/status-dot";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import {
  resumeCarrierIntelMonitoring,
  type CarrierIntelMonitoringStatus,
} from "@/lib/graphql/carrier-intel-settings";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { PlayIcon, SettingsIcon } from "lucide-react";
import { Link } from "react-router";
import { toast } from "sonner";
import { integrationSettingsPath, monitoringHealth } from "./monitoring-health";

export type ProviderStatusProps = {
  status: CarrierIntelMonitoringStatus;
  canManage: boolean;
};

export function ProviderStatus({ status, canManage }: ProviderStatusProps) {
  const t = useT();
  const now = useNowSeconds();
  const queryClient = useQueryClient();
  const health = monitoringHealth(status);
  const providerName = status.provider.provider
    ? carrierIntelProviderLabel(status.provider.provider)
    : t("No provider");

  const resume = useApiMutation({
    mutationFn: resumeCarrierIntelMonitoring,
    resourceName: "Carrier monitoring",
    onSuccess: async () => {
      toast.success(t("Monitoring resumed"), {
        description: t("Paused feeds poll again on their next scheduled run."),
      });
      await queryClient.invalidateQueries({
        queryKey: queries.carrierIntelSettings.monitoringStatus().queryKey,
      });
    },
  });

  let tone: StatusTone;
  let detail: string;
  switch (health.state) {
    case "disconnected":
      tone = "neutral";
      detail = t("Not connected");
      break;
    case "paused":
      tone = "medium";
      detail = health.pausedReason ? t("Paused: {0}", health.pausedReason) : t("Paused");
      break;
    case "failing":
      tone = "critical";
      detail = t("Last poll failed");
      break;
    case "healthy":
      tone = "success";
      detail = health.lastSuccessAt
        ? t("Connected · polled {0}", formatRelativeTime(health.lastSuccessAt - now))
        : t("Connected · waiting for the first poll");
      break;
  }

  const tooltip =
    health.state === "failing"
      ? health.lastError
      : health.state === "paused" && health.pausedAt
        ? t("Paused since {0}", formatUnixDateTimeMedium(health.pausedAt))
        : health.lastSuccessAt
          ? t("Last successful poll {0}", formatUnixDateTimeMedium(health.lastSuccessAt))
          : null;

  const summary = (
    <span className="flex min-w-0 items-center gap-2 text-sm">
      <StatusDot tone={tone} />
      <span className="font-medium">{providerName}</span>
      <span className="text-muted-foreground max-w-72 truncate text-xs">{detail}</span>
    </span>
  );

  return (
    <div className="flex flex-wrap items-center gap-2" aria-label={t("Provider status")}>
      {tooltip ? (
        <Tooltip>
          <TooltipTrigger render={<span className="flex min-w-0 cursor-default" />}>
            {summary}
          </TooltipTrigger>
          <TooltipContent className="max-w-sm">{tooltip}</TooltipContent>
        </Tooltip>
      ) : (
        summary
      )}
      {health.state === "paused" && canManage ? (
        <Button
          type="button"
          variant="outline"
          className="h-8 text-xs"
          isLoading={resume.isPending}
          loadingText={t("Resuming…")}
          onClick={() => resume.mutate()}
        >
          <PlayIcon className="size-3.5" />
          {t("Resume")}
        </Button>
      ) : null}
      <Button
        variant="ghost"
        className="h-8 text-xs"
        nativeButton={false}
        render={<Link to={integrationSettingsPath(status.provider.provider)} />}
      >
        <SettingsIcon className="size-3.5" />
        {t("Settings")}
      </Button>
    </div>
  );
}
