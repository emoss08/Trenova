import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  decideProfileChange,
  fetchProfileChangeRequests,
  PROFILE_CHANGE_REQUESTS_KEY,
  type ProfileChangeRequestRow,
} from "@/lib/graphql/self-service";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { changeRequestTone } from "@trenova/shared/lib/self-service";
import { Operation, Resource } from "@trenova/shared/types/permission";

type FieldChange = ProfileChangeRequestRow["changes"][number];

/** How many requests the list reads; the newest are the ones a manager acts on. */
const REQUEST_LIMIT = 20;

/** One field as before and after, so the eye lands on what moves. */
export function ChangeChip({ change }: { change: FieldChange }) {
  const t = useT();
  const shown = (value: string) => (value.trim() === "" ? t("(blank)") : value);
  return (
    <span className="inline-flex max-w-full flex-wrap items-center gap-1 rounded-md border border-border-subtle bg-sunken px-1.5 py-0.5 text-xs">
      <span className="text-muted-foreground">{t(change.label)}</span>
      <del className="text-muted-foreground tabular-nums decoration-foreground-subtle">
        {shown(change.from)}
      </del>
      <ArrowRightIcon className="size-3 shrink-0 text-muted-foreground" />
      <ins className="font-medium tabular-nums no-underline">{shown(change.to)}</ins>
    </span>
  );
}

/**
 * What the driver asked to change on their own record from Dash. A waiting request can be approved
 * here as it stands; turning one down, or approving with a note, opens the page's review pane.
 * Decided requests stay listed under it with the office's note.
 */
export function ProfileChangeList({
  workerId,
  onReview,
}: {
  workerId: string;
  onReview: (requestId: string) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.ProfileChangeRequest, Operation.Read);
  const { allowed: canApprove } = usePermission(Resource.ProfileChangeRequest, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.ProfileChangeRequest, Operation.Reject);

  const requests = useQuery({
    queryKey: [PROFILE_CHANGE_REQUESTS_KEY, workerId],
    queryFn: ({ signal }) =>
      fetchProfileChangeRequests({ workerId, limit: REQUEST_LIMIT }, { signal }),
    enabled: canRead,
  });

  const approve = useApiMutation({
    scope: { id: `profile-change-decide:${workerId}` },
    resourceName: "Request",
    mutationFn: (request: ProfileChangeRequestRow) =>
      decideProfileChange({ id: request.id, approve: true }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [PROFILE_CHANGE_REQUESTS_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: ["worker"] });
    },
  });

  if (!canRead) return null;

  const rows = requests.data ?? [];
  const pending = rows.filter((row) => row.status === "Pending");
  const decided = rows.filter((row) => row.status !== "Pending");

  return (
    <SectionPanel
      title={t("Profile changes")}
      count={pending.length}
      action={
        pending.length > 0 && (canApprove || canReject) ? (
          <Button size="xs" variant="quiet" onClick={() => onReview(pending[0].id)}>
            {t("Review with a note")}
          </Button>
        ) : null
      }
    >
      {requests.isPending ? (
        <div
          aria-busy
          aria-label={t("Loading the profile changes")}
          className="flex flex-col gap-2 p-3"
        >
          <span className="block h-3 w-1/3 ui-shimmer rounded-sm" />
          <span className="block h-5 w-2/3 ui-shimmer rounded-sm" />
        </div>
      ) : requests.isError ? (
        <ErrorState
          error={requests.error}
          layout="compact"
          title={t("The profile changes could not be loaded")}
          onRetry={() => void requests.refetch()}
        />
      ) : rows.length === 0 ? (
        <SectionPanelQuiet>{t("No changes asked for from Dash.")}</SectionPanelQuiet>
      ) : (
        <ScrollArea viewportClassName="max-h-96" maskVariant="card">
          {pending.map((request) => (
            <div
              key={request.id}
              data-testid={`profile-change-${request.id}`}
              className="flex flex-col gap-2 border-b border-border-subtle p-3 last:border-0"
            >
              <p className="text-xs text-muted-foreground tabular-nums">
                {t("Asked {0} from Dash", formatUnixDateMedium(request.submittedAt))}
              </p>
              <div className="flex flex-wrap gap-1.5">
                {request.changes.map((change) => (
                  <ChangeChip key={change.field} change={change} />
                ))}
              </div>
              {request.note ? (
                <blockquote className="border-l-2 border-border-strong py-0.5 pl-2.5 text-sm text-pretty">
                  {request.note}
                </blockquote>
              ) : null}
              {canApprove || canReject ? (
                <div className="flex justify-end gap-2">
                  {canReject ? (
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={approve.isPending}
                      onClick={() => onReview(request.id)}
                    >
                      {t("Turn down")}
                    </Button>
                  ) : null}
                  {canApprove ? (
                    <Button
                      size="sm"
                      isLoading={approve.isPending && approve.variables?.id === request.id}
                      disabled={approve.isPending}
                      onClick={() => void approve.mutateAsync(request).catch(() => undefined)}
                    >
                      {t("Approve")}
                    </Button>
                  ) : null}
                </div>
              ) : null}
            </div>
          ))}
          {decided.map((request) => {
            const tone = changeRequestTone(request.status);
            return (
              <div
                key={request.id}
                data-testid={`profile-change-${request.id}`}
                className="flex flex-col gap-1.5 border-b border-border-subtle px-3 py-2.5 last:border-0"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={tone.variant}>{t(tone.label)}</Badge>
                  <span className="text-xs text-muted-foreground tabular-nums">
                    {request.decidedAt
                      ? t(
                          "Asked {0}, decided {1}",
                          formatUnixDateMedium(request.submittedAt),
                          formatUnixDateMedium(request.decidedAt),
                        )
                      : t("Asked {0}", formatUnixDateMedium(request.submittedAt))}
                  </span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {request.changes.map((change) => (
                    <ChangeChip key={change.field} change={change} />
                  ))}
                </div>
                {request.decisionNote ? (
                  <p className="text-xs text-muted-foreground">
                    {t("Office: {0}", request.decisionNote)}
                  </p>
                ) : null}
              </div>
            );
          })}
        </ScrollArea>
      )}
    </SectionPanel>
  );
}
