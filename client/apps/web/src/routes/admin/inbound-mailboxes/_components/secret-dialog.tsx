import { CopyableSecret } from "@/components/copyable-secret";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { setInboundMailboxSigningSecret, type InboundMailbox } from "@/lib/graphql/inbox";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { toast } from "sonner";
import { generatePostmarkCredentials, postmarkSecret, type BasicCredentials } from "./webhook-url";

/**
 * Sets the secret a mailbox verifies deliveries with. It is sealed on the
 * server and never shown back.
 *
 * Resend's secret is copied out of Resend. Postmark has none to copy — the
 * credentials are ours to choose — so they are generated here, saved, and
 * shown once to put into the webhook URL.
 */
export function SecretDialog({
  mailbox,
  onClose,
  onSaved,
}: {
  mailbox: InboundMailbox | null;
  onClose: () => void;
  onSaved: () => Promise<void> | void;
}) {
  const t = useT();
  const [secret, setSecret] = useState("");
  const [generated, setGenerated] = useState<BasicCredentials | null>(null);

  const saveMutation = useApiMutation({
    mutationFn: ({ id, value }: { id: string; value: string }) =>
      setInboundMailboxSigningSecret(id, value),
    onSuccess: async () => {
      toast.success(t("Signing secret saved"));
      await onSaved();
    },
    resourceName: "Mailbox",
  });

  if (mailbox === null) {
    return null;
  }

  const close = () => {
    setSecret("");
    setGenerated(null);
    onClose();
  };

  const generate = () => {
    const credentials = generatePostmarkCredentials();
    saveMutation.mutate(
      { id: mailbox.id, value: postmarkSecret(credentials) },
      { onSuccess: () => setGenerated(credentials) },
    );
  };

  return (
    <Dialog open onOpenChange={(open) => !open && close()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Signing secret for {0}", mailbox.name)}</DialogTitle>
          <DialogDescription>
            {mailbox.provider === "Resend"
              ? t("Paste the inbound webhook's signing secret from Resend. It starts with whsec_.")
              : t(
                  "Postmark verifies deliveries with a user and password in the webhook URL. Generate new ones, then put them in the URL in Postmark.",
                )}
          </DialogDescription>
        </DialogHeader>

        {mailbox.provider === "Resend" ? (
          <form
            className="flex flex-col gap-2 py-2"
            onSubmit={(event) => {
              event.preventDefault();
              saveMutation.mutate({ id: mailbox.id, value: secret.trim() }, { onSuccess: close });
            }}
          >
            <Label htmlFor="mailbox-secret">{t("Signing secret")}</Label>
            <Input
              id="mailbox-secret"
              type="password"
              autoComplete="off"
              value={secret}
              onChange={(event) => setSecret(event.target.value)}
              placeholder="whsec_…"
            />
            <DialogFooter className="pt-2">
              <Button type="button" variant="outline" onClick={close}>
                {t("Cancel")}
              </Button>
              <Button
                type="submit"
                disabled={secret.trim() === ""}
                isLoading={saveMutation.isPending}
                loadingText={t("Saving…")}
              >
                {t("Save secret")}
              </Button>
            </DialogFooter>
          </form>
        ) : generated === null ? (
          <DialogFooter className="pt-2">
            <Button type="button" variant="outline" onClick={close}>
              {t("Cancel")}
            </Button>
            <Button
              onClick={generate}
              isLoading={saveMutation.isPending}
              loadingText={t("Saving…")}
            >
              {mailbox.hasSigningSecret ? t("Replace the credentials") : t("Generate credentials")}
            </Button>
          </DialogFooter>
        ) : (
          <>
            <CopyableSecret
              value={postmarkSecret(generated)}
              title={t("Copy these credentials now")}
              description={t(
                "Put them in front of the host in Postmark's webhook URL, as https://user:password@host/…. They are saved sealed and cannot be shown again; the old ones stop working now.",
              )}
            />
            <DialogFooter className="pt-2">
              <Button onClick={close}>{t("I have copied them")}</Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
