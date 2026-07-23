import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  fetchWorkerPortalStatus,
  inviteWorkerToPortal,
  revokeWorkerPortalAccess,
  type PortalInvitationRow,
} from "@/lib/graphql/driver-portal";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, CopyIcon, SmartphoneIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

function formatDate(unix?: number | null): string {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

const invitationStatusVariants: Record<string, React.ComponentProps<typeof Badge>["variant"]> = {
  Pending: "warning",
  Accepted: "active",
  Revoked: "inactive",
};

export default function WorkerPortalTab({ workerId }: { workerId: string }) {
  const queryClient = useQueryClient();
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteUrl, setInviteUrl] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [confirmRevoke, setConfirmRevoke] = useState(false);

  const status = useQuery({
    queryKey: ["worker-portal-status", workerId],
    queryFn: () => fetchWorkerPortalStatus(workerId),
    enabled: workerId.length > 0,
  });

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ["worker-portal-status", workerId] });

  const invite = useMutation({
    mutationFn: () => inviteWorkerToPortal({ workerId, email: inviteEmail.trim() || undefined }),
    onSuccess: async (result) => {
      setInviteUrl(result.inviteUrl);
      setInviteEmail("");
      toast.success(
        result.emailSent
          ? `Invitation emailed to ${result.invitation.email}`
          : "Invitation created — no email provider is configured, share the link below",
      );
      await invalidate();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to send invitation"),
  });

  const revoke = useMutation({
    mutationFn: () => revokeWorkerPortalAccess(workerId),
    onSuccess: async () => {
      toast.success("Portal access revoked");
      setConfirmRevoke(false);
      setInviteUrl(null);
      await invalidate();
    },
    onError: (error: Error) => toast.error(error.message || "Failed to revoke access"),
  });

  const copyInviteUrl = async () => {
    if (!inviteUrl) return;
    await navigator.clipboard.writeText(inviteUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  if (status.isPending) {
    return (
      <div className="flex flex-col gap-3">
        <Skeleton className="h-24 w-full rounded-lg" />
        <Skeleton className="h-32 w-full rounded-lg" />
      </div>
    );
  }

  const data = status.data;
  if (!data) {
    return (
      <p className="py-8 text-center text-sm text-muted-foreground">
        Portal status could not be loaded.
      </p>
    );
  }

  const hasPending = Boolean(data.pendingInvitation);

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-lg border border-border p-4">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <SmartphoneIcon className="size-4 text-muted-foreground" />
            <p className="text-sm font-semibold">Dash access</p>
          </div>
          {data.linked ? (
            <Badge variant="active">Linked</Badge>
          ) : hasPending ? (
            <Badge variant="warning">Invited</Badge>
          ) : (
            <Badge variant="secondary">Not set up</Badge>
          )}
        </div>

        {data.linked && data.portalUser ? (
          <div className="mt-3 flex flex-col gap-1 text-sm">
            <p className="font-medium">{data.portalUser.name}</p>
            <p className="text-xs text-muted-foreground">{data.portalUser.emailAddress}</p>
            <p className="text-xs text-muted-foreground">
              Last sign-in: {formatDate(data.portalUser.lastLoginAt)}
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-2 w-fit text-destructive"
              onClick={() => setConfirmRevoke(true)}
            >
              Revoke access
            </Button>
          </div>
        ) : hasPending && data.pendingInvitation ? (
          <div className="mt-3 flex flex-col gap-1 text-sm">
            <p className="text-xs text-muted-foreground">
              Invitation sent to{" "}
              <span className="font-medium text-foreground">{data.pendingInvitation.email}</span> —
              expires {formatDate(data.pendingInvitation.expiresAt)}.
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-2 w-fit"
              disabled={revoke.isPending}
              onClick={() => revoke.mutate()}
            >
              {revoke.isPending ? "Revoking..." : "Cancel invitation"}
            </Button>
          </div>
        ) : (
          <div className="mt-3 flex flex-col gap-2">
            <p className="text-xs text-muted-foreground">
              Invite this driver to Dash so they can see their loads, settlement statements, pay
              history, and raise pay questions from their phone.
            </p>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="portal-invite-email">Email (optional override)</Label>
              <Input
                id="portal-invite-email"
                type="email"
                placeholder="Defaults to the worker's email on file"
                value={inviteEmail}
                onChange={(event) => setInviteEmail(event.target.value)}
              />
            </div>
            <Button
              size="sm"
              className="w-fit"
              disabled={invite.isPending}
              onClick={() => invite.mutate()}
            >
              {invite.isPending ? "Sending..." : "Send invitation"}
            </Button>
          </div>
        )}

        {inviteUrl ? (
          <div className="mt-3 flex items-center gap-2 rounded-md border border-border bg-muted/40 p-2">
            <p className="min-w-0 flex-1 truncate font-mono text-xs">{inviteUrl}</p>
            <Button variant="ghost" size="sm" className="h-7 px-2" onClick={copyInviteUrl}>
              {copied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
            </Button>
          </div>
        ) : null}
      </div>

      {data.invitations.length > 0 ? (
        <div className="rounded-lg border border-border">
          <p className="px-4 pt-3 text-xs font-medium text-muted-foreground uppercase">
            Invitation history
          </p>
          <ul className="divide-y divide-border">
            {data.invitations.map((invitation: PortalInvitationRow) => (
              <li
                key={invitation.id}
                className="flex items-center justify-between gap-2 px-4 py-2.5"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm">{invitation.email}</p>
                  <p className="text-xs text-muted-foreground">
                    Sent {formatDate(invitation.createdAt)}
                    {invitation.invitedBy ? ` by ${invitation.invitedBy.name}` : ""}
                    {invitation.acceptedAt
                      ? ` · accepted ${formatDate(invitation.acceptedAt)}`
                      : ""}
                  </p>
                </div>
                <Badge variant={invitationStatusVariants[invitation.status] ?? "secondary"}>
                  {invitation.status}
                </Badge>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      <AlertDialog open={confirmRevoke} onOpenChange={setConfirmRevoke}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Revoke portal access?</AlertDialogTitle>
            <AlertDialogDescription>
              The driver&apos;s Dash login is deactivated immediately and any pending invitations
              are canceled. Their settlement history stays intact, and you can re-invite them later.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep access</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-white hover:bg-destructive/90"
              disabled={revoke.isPending}
              onClick={(event) => {
                event.preventDefault();
                revoke.mutate();
              }}
            >
              {revoke.isPending ? "Revoking..." : "Revoke access"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
