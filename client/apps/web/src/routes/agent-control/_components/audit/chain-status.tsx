import { KPI_STRIP_CELL_CLASS, KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { verifyAIAuditChain, type AIAuditChainStatus } from "@/lib/graphql/ai-audit";
import { queries } from "@/lib/queries";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import { CircleAlertIcon, ShieldCheckIcon, TriangleAlertIcon } from "lucide-react";
import { useEffect, useState } from "react";
import {
  VERIFY_POLL_MS,
  VERIFY_WAIT_MS,
  isVerificationPending,
  verificationLabel,
  verificationTone,
  type VerificationRequest,
} from "./audit-model";

/**
 * The chain the trail is written into: whether it is signed, how far it is
 * sealed, and what its last check found, with a way to check it now. A check
 * runs in the background; the status says so until its result is stored.
 */
export function ChainStatusStrip() {
  const t = useT();
  const queryClient = useQueryClient();
  const statusQuery = queries.aiAudit.chainStatus();
  const [request, setRequest] = useState<VerificationRequest | null>(null);
  const [now] = useState(() => Date.now());

  const chain = useQuery({
    ...statusQuery,
    refetchInterval: (query) =>
      isVerificationPending(request, query.state.data ?? null) ? VERIFY_POLL_MS : false,
  });

  // A check that never reports back stops being waited on, so the button
  // comes back rather than spinning for the rest of the visit.
  useEffect(() => {
    if (request === null) {
      return;
    }
    const timer = window.setTimeout(() => setRequest(null), VERIFY_WAIT_MS);
    return () => window.clearTimeout(timer);
  }, [request]);

  const verify = useMutation({
    mutationFn: verifyAIAuditChain,
    onMutate: () => setRequest(null),
    onSuccess: (status) => {
      setRequest({ baseline: status.lastVerifiedAt ?? null });
      queryClient.setQueryData(statusQuery.queryKey, status);
    },
  });

  if (chain.isError) {
    return (
      <Alert variant="destructive" size="sm">
        <CircleAlertIcon />
        <AlertDescription>
          {t("The audit chain's status could not be loaded. Try again shortly.")}
        </AlertDescription>
      </Alert>
    );
  }

  if (!chain.data) {
    return <Skeleton className="h-16" aria-busy />;
  }

  const status = chain.data;
  const pending = isVerificationPending(request, status);

  return (
    <div className="flex flex-col gap-2">
      <KpiStrip aria-label={t("Audit chain")}>
        <KpiStripItem
          label={t("Chain")}
          value={status.signed ? t("Signed") : t("Unsigned")}
          sub={
            status.signed
              ? t("Key {0}", status.activeKeyId ?? "—")
              : t("Plain SHA-256; no signing key is configured")
          }
          tone={status.signed ? "success" : "warning"}
        />
        <KpiStripItem
          label={t("Sealed through")}
          value={status.lastSeq > 0 ? `#${status.sealedThroughSeq.toLocaleString()}` : "—"}
          sub={
            status.lastSeq > 0
              ? t("of {0} recorded", status.lastSeq.toLocaleString())
              : t("Nothing recorded yet")
          }
        />
        <KpiStripItem
          label={t("Last verified")}
          value={verificationLabel(t, status.lastVerificationStatus)}
          sub={lastVerifiedSub(t, status, now)}
          tone={verificationTone(status.lastVerificationStatus)}
        />
        <div className={cn(KPI_STRIP_CELL_CLASS, "flex flex-col justify-center gap-1")}>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => verify.mutate()}
            disabled={pending || verify.isPending}
            isLoading={verify.isPending}
            loadingText={t("Starting the check…")}
            className="self-start"
          >
            <ShieldCheckIcon className="size-3.5" />
            {pending ? t("Verifying…") : t("Verify now")}
          </Button>
          <span className="text-foreground-muted text-xs">
            {pending
              ? t("The result appears here when the check finishes.")
              : t("Checked every night on its own.")}
          </span>
        </div>
      </KpiStrip>
      {verify.isError ? (
        <Alert variant="destructive" size="sm">
          <CircleAlertIcon />
          <AlertDescription>
            {graphQLErrorMessage(verify.error, t("The check could not be started."))}
          </AlertDescription>
        </Alert>
      ) : null}
      {status.lastVerificationStatus === "Mismatch" ? (
        <Alert variant="destructive" size="sm">
          <TriangleAlertIcon />
          <AlertDescription>
            {status.failedSeq != null
              ? t(
                  "The trail no longer matches its chain at #{0}: {1}. The failure is in the audit log, and everyone who reads the trail was told.",
                  status.failedSeq.toLocaleString(),
                  status.detail ?? t("no detail was recorded"),
                )
              : t(
                  "The trail no longer matches its chain: {0}. The failure is in the audit log, and everyone who reads the trail was told.",
                  status.detail ?? t("no detail was recorded"),
                )}
          </AlertDescription>
        </Alert>
      ) : null}
      {status.lastVerificationStatus === "KeyMissing" ? (
        <Alert variant="warning" size="sm">
          <TriangleAlertIcon />
          <AlertDescription>
            {t(
              "A row names a signing key that is no longer configured, so the chain cannot be checked past it. Put the key back, then verify again.",
            )}
          </AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}

function lastVerifiedSub(
  t: ReturnType<typeof useT>,
  status: AIAuditChainStatus,
  now: number,
): string {
  if (status.lastVerifiedAt == null) {
    return t("Never checked");
  }

  const at = formatUnixDateTimeShort(status.lastVerifiedAt);
  const ageDays = Math.floor((now / 1000 - status.lastVerifiedAt) / 86_400);
  return ageDays > 1
    ? t("{0} ({1} days ago), through #{2}", at, ageDays, status.lastVerifiedSeq.toLocaleString())
    : t("{0}, through #{1}", at, status.lastVerifiedSeq.toLocaleString());
}
