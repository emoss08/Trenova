import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { rotateInboundMailboxToken, type InboundMailbox } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PlusIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { CredentialsDialog } from "./_components/credentials-dialog";
import { MailboxCard } from "./_components/mailbox-card";
import { MailboxFormDialog, type MailboxCreated } from "./_components/mailbox-form-dialog";
import { SecretDialog } from "./_components/secret-dialog";

export const prefetch: RoutePrefetch = () => [queries.inbox.mailboxes()];

/**
 * The addresses the system listens on.
 *
 * Setting one up is three things — the address, where the provider posts,
 * and the secret that proves a delivery came from the provider — and this
 * page does them in that order, showing the URL the one time it can be shown.
 */
export function InboundMailboxesPage() {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canCreate } = usePermission(Resource.InboundMailbox, Operation.Create);
  const { allowed: canEdit } = usePermission(Resource.InboundMailbox, Operation.Update);

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<InboundMailbox | null>(null);
  const [secretFor, setSecretFor] = useState<InboundMailbox | null>(null);
  const [rotating, setRotating] = useState<InboundMailbox | null>(null);
  const [shown, setShown] = useState<MailboxCreated | null>(null);

  const mailboxesQuery = useQuery(queries.inbox.mailboxes());
  const mailboxes = mailboxesQuery.data ?? [];

  const refresh = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: queries.inbox._def });
  }, [queryClient]);

  const rotateMutation = useApiMutation({
    mutationFn: (id: string) => rotateInboundMailboxToken(id),
    onSuccess: async (credentials) => {
      toast.success(t("A new webhook URL is ready"));
      setRotating(null);
      setShown({ credentials });
      await refresh();
    },
    resourceName: "Mailbox",
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Inbound mailboxes"),
        description: t(
          "Addresses the system listens on, where each provider posts, and how much each is trusted to do without a person",
        ),
        actions: canCreate ? (
          <Button
            onClick={() => {
              setEditing(null);
              setFormOpen(true);
            }}
          >
            <PlusIcon className="size-4" />
            {t("New mailbox")}
          </Button>
        ) : undefined,
      }}
    >
      {mailboxesQuery.isLoading ? (
        <div className="grid gap-3 lg:grid-cols-2">
          <Skeleton className="h-44" />
          <Skeleton className="h-44" />
        </div>
      ) : mailboxes.length === 0 ? (
        <EmptySheet
          className="my-10"
          title={t("No mailboxes yet")}
          description={t(
            "Create one for each address customers and carriers send to — tenders, PODs, invoices — and point your email provider's inbound webhook at it.",
          )}
          sketch={
            <div className="flex flex-col gap-3">
              <GhostLine className="w-1/2" />
              <GhostLine className="w-2/3" />
            </div>
          }
          action={
            canCreate ? (
              <Button onClick={() => setFormOpen(true)}>
                <PlusIcon className="size-4" />
                {t("New mailbox")}
              </Button>
            ) : undefined
          }
        />
      ) : (
        <div className="grid gap-3 lg:grid-cols-2">
          {mailboxes.map((mailbox) => (
            <MailboxCard
              key={mailbox.id}
              mailbox={mailbox}
              canEdit={canEdit}
              onEdit={() => {
                setEditing(mailbox);
                setFormOpen(true);
              }}
              onRotate={() => setRotating(mailbox)}
              onSecret={() => setSecretFor(mailbox)}
            />
          ))}
        </div>
      )}

      <MailboxFormDialog
        open={formOpen}
        mailbox={editing}
        onClose={() => setFormOpen(false)}
        onCreated={(created) => {
          setFormOpen(false);
          setShown(created);
        }}
        onSaved={refresh}
      />
      <SecretDialog mailbox={secretFor} onClose={() => setSecretFor(null)} onSaved={refresh} />
      <CredentialsDialog
        credentials={shown?.credentials ?? null}
        postmark={shown?.postmark}
        onClose={() => setShown(null)}
      />

      <AlertDialog open={rotating !== null} onOpenChange={(open) => !open && setRotating(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Rotate the webhook URL?")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "The current URL stops working immediately. Mail will be refused until the provider is pointed at the new one.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={rotateMutation.isPending}
              onClick={() => rotating !== null && rotateMutation.mutate(rotating.id)}
            >
              {t("Rotate")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </PageLayout>
  );
}
