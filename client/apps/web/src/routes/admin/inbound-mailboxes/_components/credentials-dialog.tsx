import { CopyableSecret } from "@/components/copyable-secret";
import type { InboundMailboxCredentials } from "@/lib/graphql/inbox";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { webhookUrl, type BasicCredentials } from "./webhook-url";

/**
 * The one look anybody gets at a mailbox's webhook URL.
 *
 * The token in it is stored only as a hash, so closing this without copying
 * the URL means rotating for a new one. For Postmark the credentials the
 * mailbox was just given are already in the URL, so pasting it into Postmark
 * is the whole of the setup.
 */
export function CredentialsDialog({
  credentials,
  postmark,
  onClose,
}: {
  credentials: InboundMailboxCredentials | null;
  postmark?: BasicCredentials;
  onClose: () => void;
}) {
  const t = useT();

  if (credentials === null) {
    return null;
  }

  const provider = credentials.mailbox.provider;
  const url = webhookUrl(API_BASE_URL, window.location.origin, credentials.webhookPath, postmark);

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{t("Point {0} at this address", provider)}</DialogTitle>
          <DialogDescription>
            {t("{0} will post {1}'s mail here.", provider, credentials.mailbox.address)}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-3 py-2">
          <CopyableSecret
            value={url}
            title={t("Copy the webhook URL now")}
            description={t(
              "It carries the mailbox's token, which is stored only as a hash and cannot be shown again. Lose it and you rotate for a new one.",
            )}
          />

          {provider === "Resend" ? (
            <Alert size="sm" variant="info">
              <AlertDescription>
                {credentials.mailbox.hasSigningSecret
                  ? t(
                      "In Resend, set this as the inbound webhook. The signing secret is already saved, so deliveries are verified from the first one.",
                    )
                  : t(
                      "In Resend, set this as the inbound webhook, then copy the webhook's signing secret (it starts with whsec_) into Set signing secret. Until then the mailbox refuses every delivery.",
                    )}
              </AlertDescription>
            </Alert>
          ) : (
            <Alert size="sm" variant="info">
              <AlertDescription>
                {postmark
                  ? t(
                      "In Postmark, set this as the server's inbound webhook URL. The user and password in it are the mailbox's credentials; they are saved, and Postmark sends them with every delivery.",
                    )
                  : t(
                      "In Postmark, set this as the inbound webhook URL with the mailbox's user and password in front of the host (https://user:password@…). Until credentials are set the mailbox refuses every delivery.",
                    )}
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button onClick={onClose}>{t("I have copied it")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
