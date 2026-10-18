import { PortalCodeForm } from "@/components/assistant/portal-code-form";
import { PortalLiveBrowser } from "@/components/assistant/portal-sign-in";
import { ExternalLink } from "@/components/link";
import type { PortalRunResultView } from "@/lib/portal-run";
import { apiService } from "@/services/api";
import type { PortalRecording } from "@/types/agent-extension";
import { useMutation } from "@tanstack/react-query";
import { PlayCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

/**
 * How a portal task stands and what it found, wherever a person follows one: while it runs, a
 * way to watch the live browser; while it waits on a sign-in code, where to enter it or sign in
 * by hand; once it ends, what the portal said (written outside the organization), why it did
 * not finish, every page it reached and its recording.
 */
export function PortalRunResult({
  run,
  canEnterCode,
  className,
}: {
  run: PortalRunResultView;
  /** Whether this person may enter the code or take over the browser; the server checks again. */
  canEnterCode: boolean;
  className?: string;
}) {
  const t = useT();

  return (
    <div className={cn("flex flex-col gap-3 text-sm", className)}>
      {run.status === "Running" && run.hasRecording && canEnterCode && (
        <div className="flex flex-col gap-1.5">
          <p className="text-muted-foreground text-xs">
            {t("The browser is working on it. You can watch it as it goes.")}
          </p>
          <PortalLiveBrowser runId={run.runId} label={t("Watch the browser")} />
        </div>
      )}
      {run.status === "AwaitingCode" && run.codeExpiresAt !== null && canEnterCode && (
        <PortalCodeForm
          className="bg-background rounded-md border p-2"
          runId={run.runId}
          prompt={run.codePrompt}
          deadline={formatUnixTime(run.codeExpiresAt)}
          attempts={run.codeAttempts}
        />
      )}
      {run.answer !== "" && run.status !== "Strayed" && (
        <section aria-label={t("What the portal said")} className="flex flex-col gap-1">
          <span className="text-muted-foreground text-xs">{t("What the portal said")}</span>
          <p className="whitespace-pre-wrap">{run.answer}</p>
          <span className="text-muted-foreground text-xs">
            {t("Read from the portal, outside your organization. Check anything you act on.")}
          </span>
        </section>
      )}
      {run.failure !== "" && (
        <Alert size="sm" variant={run.status === "Strayed" ? "warning" : "destructive"}>
          <AlertDescription>{run.failure}</AlertDescription>
        </Alert>
      )}
      {run.visitedUrls.length > 0 && (
        <div className="flex flex-col gap-1">
          <span className="text-muted-foreground text-xs">
            {t("{0, plural, one {# page visited} other {# pages visited}}", run.visitedUrls.length)}
          </span>
          <ul className="flex flex-col gap-0.5 font-mono text-xs">
            {run.visitedUrls.map((url, index) => {
              const offSite = run.offSiteUrls.includes(url);
              return (
                <li
                  key={`${index}-${url}`}
                  className={cn("truncate", offSite && "text-danger")}
                  title={offSite ? t("Outside the site's addresses") : url}
                >
                  {url}
                </li>
              );
            })}
          </ul>
        </div>
      )}
      {run.hasRecording && run.status !== "Running" && run.status !== "AwaitingCode" && (
        <PortalRecordingLink runId={run.runId} />
      )}
    </div>
  );
}

/** The task's recording, asked for when wanted: its address is signed and short-lived. */
function PortalRecordingLink({ runId }: { runId: string }) {
  const t = useT();
  const fetchRecording = useMutation<PortalRecording, Error>({
    mutationFn: () => apiService.agentExtensionService.portalRunRecording(runId),
  });
  const recording = fetchRecording.data;
  const link = recording?.shareUrl || recording?.recordingUrl || "";

  return (
    <div className="flex flex-wrap items-center gap-2">
      {link ? (
        <ExternalLink href={link}>{t("Open the recording")}</ExternalLink>
      ) : (
        <Button
          type="button"
          size="sm"
          variant="outline"
          isLoading={fetchRecording.isPending}
          loadingText={t("Recording")}
          onClick={() => fetchRecording.mutate()}
        >
          <PlayCircleIcon className="size-3.5" />
          {t("Recording")}
        </Button>
      )}
      {fetchRecording.isError && (
        <Alert size="sm" variant="destructive">
          <AlertDescription>
            {fetchRecording.error instanceof Error && fetchRecording.error.message
              ? fetchRecording.error.message
              : t("The recording could not be found.")}
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}
