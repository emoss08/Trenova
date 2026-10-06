import { useApiMutation } from "@/hooks/use-api-mutation";
import { setInboundMailboxApiKey, type InboundMailbox } from "@/lib/graphql/inbox";
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
import { useEffect, useRef, useState } from "react";
import { toast } from "sonner";

/**
 * Sets the key a Resend mailbox reads message content with. Resend's webhook
 * carries only a message's sender, recipients and subject; the body and the
 * attachments are fetched afterwards with this key. It is sealed on the server
 * and never shown back.
 */
export function ApiKeyDialog({
  mailbox,
  onClose,
  onSaved,
}: {
  mailbox: InboundMailbox | null;
  onClose: () => void;
  onSaved: () => Promise<void> | void;
}) {
  const t = useT();
  const [apiKey, setApiKey] = useState("");
  // A save can finish after its dialog was dismissed and another mailbox's
  // opened; only the dialog for the mailbox that was saved closes on success.
  const shownId = useRef<string | null>(null);
  useEffect(() => {
    shownId.current = mailbox?.id ?? null;
  }, [mailbox]);

  const saveMutation = useApiMutation({
    mutationFn: ({ id, value }: { id: string; value: string }) =>
      setInboundMailboxApiKey(id, value),
    onSuccess: async () => {
      toast.success(t("API key saved"));
      await onSaved();
    },
    resourceName: "Mailbox",
  });

  if (mailbox === null) {
    return null;
  }

  const close = () => {
    setApiKey("");
    onClose();
  };

  return (
    <Dialog open onOpenChange={(open) => !open && close()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("API key for {0}", mailbox.name)}</DialogTitle>
          <DialogDescription>
            {t(
              "Resend sends only the sender and subject to the webhook. Create a full access API key in Resend and paste it here so the body and attachments of each message can be read. It starts with re_.",
            )}
          </DialogDescription>
        </DialogHeader>

        <form
          className="flex flex-col gap-2 py-2"
          onSubmit={(event) => {
            event.preventDefault();
            saveMutation.mutate(
              { id: mailbox.id, value: apiKey.trim() },
              {
                onSuccess: (_saved, variables) => {
                  if (shownId.current === variables.id) {
                    close();
                  }
                },
              },
            );
          }}
        >
          <Label htmlFor="mailbox-api-key">{t("API key")}</Label>
          <Input
            id="mailbox-api-key"
            type="password"
            autoComplete="off"
            value={apiKey}
            onChange={(event) => setApiKey(event.target.value)}
            placeholder="re_…"
          />
          <p className="text-foreground-subtle text-xs">
            {t(
              "Mail that arrives after the key is saved is read in full. A sending access key cannot read received mail.",
            )}
          </p>
          <DialogFooter className="pt-2">
            <Button type="button" variant="outline" onClick={close}>
              {t("Cancel")}
            </Button>
            <Button
              type="submit"
              disabled={apiKey.trim() === ""}
              isLoading={saveMutation.isPending}
              loadingText={t("Saving…")}
            >
              {t("Save API key")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
