import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchWorkerPolicyAcknowledgements,
  POLICY_ACKNOWLEDGEMENTS_KEY,
} from "@/lib/graphql/self-service";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { FileCheck02Icon, FileClockIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";

/**
 * Every policy version this worker has signed. A signature on a superseded version stays listed:
 * it is still a fact about what they had agreed to at the time.
 */
export function PolicyAcknowledgements({ workerId }: { workerId: string }) {
  const t = useT();
  const rt = useRichT();
  const { allowed: canRead } = usePermission(Resource.WorkerPolicy, Operation.Read);

  const acks = useQuery({
    queryKey: [POLICY_ACKNOWLEDGEMENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPolicyAcknowledgements(workerId, { signal }),
    enabled: canRead,
  });

  if (!canRead) return null;
  const rows = acks.data ?? [];

  return (
    <SectionPanel title={t("Policies signed")} count={rows.length}>
      {acks.isPending ? (
        <div
          aria-busy
          aria-label={t("Loading the policies signed")}
          className="flex flex-col gap-2 p-3"
        >
          <span className="block h-3 w-1/2 ui-shimmer rounded-sm" />
          <span className="block h-3 w-2/5 ui-shimmer rounded-sm" />
        </div>
      ) : acks.isError ? (
        <ErrorState
          error={acks.error}
          layout="compact"
          title={t("The policies signed could not be loaded")}
          onRetry={() => void acks.refetch()}
        />
      ) : rows.length === 0 ? (
        <SectionPanelQuiet>
          {t("Nothing signed yet. Policies are signed from Dash, under Profile.")}
        </SectionPanelQuiet>
      ) : (
        <ScrollArea viewportClassName="max-h-72" maskVariant="card">
          <ul className="flex flex-col divide-y divide-border-subtle px-1.5 py-1">
            {rows.map((ack) => {
              const superseded = Boolean(
                ack.policy && ack.policy.versionLabel !== ack.versionLabel,
              );
              const Icon = superseded ? FileClockIcon : FileCheck02Icon;
              return (
                <li
                  key={ack.id}
                  className="grid grid-cols-[0.875rem_minmax(0,1fr)] items-start gap-2.5 p-2"
                >
                  <Icon className="mt-0.5 size-3.5 text-muted-foreground" />
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                      <span className="text-sm font-medium">
                        {ack.policy?.title ?? t("Policy")}
                      </span>
                      <Badge variant="neutral" className="font-mono">
                        {t("v{0}", ack.versionLabel)}
                      </Badge>
                      {superseded ? (
                        <Badge variant="warning" dot>
                          {t("Superseded by v{0}", ack.policy?.versionLabel)}
                        </Badge>
                      ) : null}
                    </div>
                    <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
                      {ack.signatureName
                        ? rt(
                            "Signed {0} as <sig>{1}</sig>",
                            {
                              sig: (chunk) => (
                                <span className="text-foreground italic">{chunk}</span>
                              ),
                            },
                            formatUnixDateMedium(ack.acknowledgedAt),
                            ack.signatureName,
                          )
                        : t("Read {0}", formatUnixDateMedium(ack.acknowledgedAt))}
                    </p>
                  </div>
                </li>
              );
            })}
          </ul>
        </ScrollArea>
      )}
    </SectionPanel>
  );
}
