import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { CheckIcon, Copy01Icon } from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { toast } from "@trenova/shared/components/ui/toast";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useCanReadKind } from "@trenova/shared/hooks/use-can-read-kind";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import {
  fetchWorkerPortalStatus,
  inviteWorkerToPortal,
  revokeWorkerPortalAccess,
} from "@trenova/shared/lib/graphql/driver-portal";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useId, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { FilesSkeleton } from "../files-skeleton";
import { PolicyAcknowledgements } from "./policy-acknowledgements";
import { ProfileChangeList } from "./profile-change-list";

const WORKER_PORTAL_STATUS_KEY = "worker-portal-status";

const INVITATION_STATUS_LABELS: Record<string, string> = defineLabels({
  Pending: "Pending",
  Accepted: "Accepted",
  Revoked: "Revoked",
  Expired: "Expired",
});

const INVITATION_STATUS_TONE: Record<string, "warning" | "success" | "danger" | "neutral"> = {
  Pending: "warning",
  Accepted: "success",
  Revoked: "danger",
};

type DriverPortalProps = Pick<WorkerTabProps, "workerId" | "worker" | "onOpen">;

/**
 * The driver's side of the record: whether they can sign in to Dash, the profile changes they
 * asked for there, the policies they signed, and every invitation sent. Reviewing a change opens
 * the page's profile change pane; revoking access asks first.
 */
export default function WorkerPortalTab({ workerId, worker, onOpen }: DriverPortalProps) {
  const t = useT();
  const rt = useRichT();
  const emailId = useId();
  const queryClient = useQueryClient();
  const canReadUsers = useCanReadKind("user");
  const { allowed: canView } = usePermission(Resource.DriverPortal, Operation.Read);
  const { allowed: canInvite } = usePermission(Resource.DriverPortal, Operation.Create);
  const { allowed: canRevoke } = usePermission(Resource.DriverPortal, Operation.Cancel);
  const { copy, isCopied: copied } = useCopyToClipboard();
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteUrl, setInviteUrl] = useState<string | null>(null);
  const [confirm, setConfirm] = useState<"revoke" | "cancel" | null>(null);

  const status = useQuery({
    queryKey: [WORKER_PORTAL_STATUS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerPortalStatus(workerId, { signal }),
    enabled: canView,
  });

  const invalidate = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: [WORKER_PORTAL_STATUS_KEY, workerId] }),
      queryClient.invalidateQueries({ queryKey: ["worker"] }),
    ]);

  const invite = useApiMutation({
    scope: { id: `worker-portal:${workerId}` },
    resourceName: "Invitation",
    mutationFn: () => inviteWorkerToPortal({ workerId, email: inviteEmail.trim() || undefined }),
    onSuccess: (result) => {
      setInviteUrl(result.inviteUrl ?? null);
      setInviteEmail("");
      if (!result.emailSent) {
        toast.info(t("No email provider is set up. Share the invitation link instead."));
      }
      void invalidate();
    },
  });
  const revoke = useApiMutation({
    scope: { id: `worker-portal:${workerId}` },
    resourceName: "Dash access",
    mutationFn: () => revokeWorkerPortalAccess(workerId),
    onSuccess: () => {
      setInviteUrl(null);
      void invalidate();
    },
  });

  const review = (requestId: string) =>
    onOpen({ tab: "files", view: "driver-portal", request: requestId });

  const sideColumn = (
    <div className="flex min-w-0 flex-col gap-4 text-sm">
      <ProfileChangeList workerId={workerId} onReview={review} />
      <PolicyAcknowledgements workerId={workerId} />
    </div>
  );

  if (!canView) return sideColumn;
  if (status.isPending) return <FilesSkeleton view="driver-portal" />;
  if (status.isError) {
    return (
      <ErrorState
        error={status.error}
        layout="sheet"
        title={t("Dash access could not be loaded")}
        onRetry={() => void status.refetch()}
      />
    );
  }

  const data = status.data;
  const pending = data.pendingInvitation;

  return (
    <div className="@container text-sm">
      <div className="grid items-start gap-4 @5xl:grid-cols-[minmax(0,1fr)_24rem]">
        <div className="flex min-w-0 flex-col gap-4">
          <SectionPanel title={t("Dash access")} hint={t("The driver app")}>
            {data.linked && data.portalUser ? (
              <div className="flex flex-col items-start justify-between gap-3 p-3 @xl:flex-row @xl:items-center">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                    <Badge variant="success" dot>
                      {t("Linked")}
                    </Badge>
                    {data.portalUser.emailAddress === null ? (
                      <WithheldValue kind="user" />
                    ) : (
                      <span className="text-sm font-medium">{data.portalUser.emailAddress}</span>
                    )}
                  </div>
                  {canReadUsers ? (
                    <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
                      {data.portalUser.lastLoginAt
                        ? t(
                            "Last sign-in {0}",
                            formatUnixDateTimeMedium(data.portalUser.lastLoginAt),
                          )
                        : t("Has not signed in yet")}
                    </p>
                  ) : null}
                </div>
                {canRevoke ? (
                  <Button
                    size="sm"
                    variant="outline"
                    className="text-danger-foreground"
                    onClick={() => setConfirm("revoke")}
                  >
                    {t("Revoke access")}
                  </Button>
                ) : null}
              </div>
            ) : pending ? (
              <div className="flex flex-col items-start justify-between gap-3 p-3 @xl:flex-row @xl:items-center">
                <div className="min-w-0">
                  <Badge variant="warning" dot>
                    {t("Invited")}
                  </Badge>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {rt(
                      "Sent to <b>{0}</b>. It expires {1}.",
                      { b: (chunk) => <b className="font-medium text-foreground">{chunk}</b> },
                      pending.email,
                      formatUnixDateMedium(pending.expiresAt),
                    )}
                  </p>
                </div>
                {canRevoke ? (
                  <Button size="sm" variant="outline" onClick={() => setConfirm("cancel")}>
                    {t("Cancel invitation")}
                  </Button>
                ) : null}
              </div>
            ) : canInvite ? (
              <form
                className="flex flex-col gap-2 p-3"
                onSubmit={(submitEvent) => {
                  submitEvent.preventDefault();
                  void invite.mutateAsync().catch(() => undefined);
                }}
              >
                <div className="flex items-center gap-2">
                  <Badge variant="neutral">{t("Not set up")}</Badge>
                  <p className="text-xs text-muted-foreground">
                    {t(
                      "Invite {0} to Dash to see loads, settlements and pay, and to raise pay questions from a phone.",
                      worker.firstName,
                    )}
                  </p>
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor={emailId}>{t("Email, if none is on file")}</Label>
                  <div className="flex gap-2">
                    <Input
                      id={emailId}
                      type="email"
                      placeholder={t("The worker's email on file")}
                      value={inviteEmail}
                      onChange={(changeEvent) => setInviteEmail(changeEvent.target.value)}
                      className="h-7 max-w-sm"
                    />
                    <Button type="submit" size="sm" isLoading={invite.isPending}>
                      {t("Send invitation")}
                    </Button>
                  </div>
                </div>
              </form>
            ) : (
              <SectionPanelQuiet>
                {t("{0} has not been invited to Dash.", worker.firstName)}
              </SectionPanelQuiet>
            )}
            {inviteUrl ? (
              <div className="mx-3 mb-3 flex items-center gap-2 rounded-md border border-border-subtle bg-sunken p-1.5 pl-2.5">
                <p className="min-w-0 flex-1 truncate font-mono text-xs">{inviteUrl}</p>
                <Button
                  size="icon-xs"
                  variant="ghost"
                  aria-label={t("Copy the invitation link")}
                  onClick={() => void copy(inviteUrl)}
                >
                  {copied ? (
                    <CheckIcon className="size-3.5" />
                  ) : (
                    <Copy01Icon className="size-3.5" />
                  )}
                </Button>
              </div>
            ) : null}
          </SectionPanel>
          {sideColumn}
        </div>

        <SectionPanel title={t("Invitations")} count={data.invitations.length}>
          {data.invitations.length === 0 ? (
            <SectionPanelQuiet>{t("No invitations sent.")}</SectionPanelQuiet>
          ) : (
            <ScrollArea viewportClassName="max-h-96" maskVariant="card">
              <ul className="flex flex-col divide-y divide-border-subtle px-1.5 py-1">
                {data.invitations.map((invitation) => (
                  <li key={invitation.id} className="flex items-start justify-between gap-3 p-2">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{invitation.email}</p>
                      <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
                        {[
                          invitation.invitedBy
                            ? t(
                                "Sent {0} by {1}",
                                formatUnixDateMedium(invitation.createdAt),
                                invitation.invitedBy.name,
                              )
                            : t("Sent {0}", formatUnixDateMedium(invitation.createdAt)),
                          invitation.acceptedAt
                            ? t("accepted {0}", formatUnixDateMedium(invitation.acceptedAt))
                            : null,
                        ]
                          .filter(Boolean)
                          .join(" · ")}
                      </p>
                    </div>
                    <Badge
                      variant={INVITATION_STATUS_TONE[invitation.status] ?? "neutral"}
                      dot={invitation.status !== "Revoked"}
                    >
                      {INVITATION_STATUS_LABELS[invitation.status] ?? t("Unknown")}
                    </Badge>
                  </li>
                ))}
              </ul>
            </ScrollArea>
          )}
        </SectionPanel>
      </div>

      <ConfirmDialog
        open={confirm !== null}
        onOpenChange={(next) => !next && setConfirm(null)}
        title={
          confirm === "cancel"
            ? t("Cancel this invitation?")
            : t("Revoke {0}'s Dash access?", worker.firstName)
        }
        description={
          confirm === "cancel"
            ? t("The link stops working. You can send a new invitation later.")
            : t(
                "{0} is signed out of Dash on every device and cannot see loads, pay or time off there until you invite them again. The account and its history are kept.",
                worker.firstName,
              )
        }
        confirmLabel={confirm === "cancel" ? t("Cancel invitation") : t("Revoke access")}
        cancelLabel={confirm === "cancel" ? t("Keep it") : undefined}
        pending={revoke.isPending}
        onConfirm={() => revoke.mutateAsync()}
      />
    </div>
  );
}
