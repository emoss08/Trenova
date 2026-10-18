import { ExternalLink } from "@/components/link";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { agentExtension } from "@/lib/queries/agent-extension";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import type { PortalLiveView, PortalRun } from "@/types/agent-extension";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { BrowserIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

/** Reads a portal task again wherever it is shown, after a person moved it on. */
export function useInvalidatePortalRun() {
  const queryClient = useQueryClient();

  return () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: assistant.waits._def }),
      queryClient.invalidateQueries({ queryKey: agentExtension.portalRuns._def }),
      queryClient.invalidateQueries({ queryKey: agentExtension.portalRun._def }),
    ]);
}

/**
 * The open browser a portal task runs in. Its address is signed in to the portal, so it is read
 * only when the person asks for it, and opened from a link they follow.
 */
export function PortalLiveBrowser({
  runId,
  label,
  className,
}: {
  runId: string;
  label: string;
  className?: string;
}) {
  const t = useT();
  const live = useMutation<PortalLiveView, Error>({
    mutationFn: () => apiService.agentExtensionService.portalRunLive(runId),
  });
  const address = live.data?.liveUrl ?? "";

  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      {address ? (
        <ExternalLink href={address}>{t("Open the live browser")}</ExternalLink>
      ) : (
        <Button
          type="button"
          size="sm"
          variant="outline"
          isLoading={live.isPending}
          loadingText={label}
          onClick={() => live.mutate()}
        >
          <BrowserIcon className="size-3.5" />
          {label}
        </Button>
      )}
      {live.isError && (
        <Alert size="sm" variant="destructive">
          <AlertDescription>
            {live.error instanceof Error && live.error.message
              ? live.error.message
              : t("The live browser could not be opened.")}
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}

/**
 * Signing in by hand when a relayed code is too slow or the portal wants more than a code: the
 * person opens the live browser, signs in there, and says so, and the task carries on from the
 * page the browser is on.
 */
export function PortalSignInYourself({ runId }: { runId: string }) {
  const t = useT();
  const invalidate = useInvalidatePortalRun();
  const { mutate, isPending } = useApiMutation<PortalRun, void>({
    resourceName: t("Portal task"),
    mutationFn: () => apiService.agentExtensionService.confirmPortalSignIn(runId),
    onSuccess: invalidate,
  });

  return (
    <div className="flex flex-col gap-1.5">
      <p className="text-muted-foreground text-xs">
        {t("Or sign in yourself in the live browser, then tell the task to carry on.")}
      </p>
      <div className="flex flex-wrap items-center gap-2">
        <PortalLiveBrowser runId={runId} label={t("Live browser")} />
        <Button
          type="button"
          size="sm"
          variant="outline"
          isLoading={isPending}
          loadingText={t("Carrying on")}
          onClick={() => mutate()}
        >
          {t("I've signed in")}
        </Button>
      </div>
    </div>
  );
}
