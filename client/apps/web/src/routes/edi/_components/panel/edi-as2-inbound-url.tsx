import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { usePermission } from "@/hooks/use-permission";
import type { EDICommunicationProfileRow } from "@/lib/graphql/edi-table";
import { webhookUrl } from "@/routes/admin/inbound-mailboxes/_components/webhook-url";
import { ediService } from "@/services/edi";
import { CheckIcon, ClipboardIcon } from "@trenova/shared/components/icons";
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
import { FormSection } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { invalidateEDICommunicationProfiles } from "./edi-panel-invalidation";

/**
 * The URL this profile's partner posts AS2 messages to. It belongs to the one
 * profile, so it is what a partner is given during onboarding, and rotating it
 * retires the old one at once.
 */
export function AS2InboundURLSection({ profile }: { profile: EDICommunicationProfileRow | null }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { copy, isCopied } = useCopyToClipboard();
  const { allowed: canUpdate } = usePermission(Resource.EDI, Operation.Update);
  const [rotatedPath, setRotatedPath] = useState("");
  const [confirming, setConfirming] = useState(false);

  const rotateMutation = useApiMutation({
    mutationFn: (profileId: string) => ediService.rotateInboundUrl(profileId),
    onSuccess: async (rotated) => {
      setRotatedPath(rotated.as2InboundPath ?? "");
      setConfirming(false);
      await invalidateEDICommunicationProfiles(queryClient);
    },
    resourceName: "Communication profile",
  });

  const path = rotatedPath || profile?.as2InboundPath || "";
  const url = path ? webhookUrl(API_BASE_URL, window.location.origin, path) : "";

  return (
    <FormSection title={t("Inbound URL")}>
      {url ? (
        <div className="flex flex-col gap-2">
          <p className="text-muted-foreground text-xs">
            {t(
              "Give this URL to the partner. It is the only address that accepts their AS2 messages for this profile.",
            )}
          </p>
          <code className="bg-muted text-foreground block rounded-md px-2 py-1.5 font-mono text-xs break-all">
            {url}
          </code>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => void copy(url, { withToast: true })}
            >
              {isCopied ? <CheckIcon /> : <ClipboardIcon />}
              {isCopied ? t("Copied") : t("Copy")}
            </Button>
            {canUpdate && profile ? (
              <Button type="button" variant="outline" size="sm" onClick={() => setConfirming(true)}>
                {t("Rotate URL")}
              </Button>
            ) : null}
          </div>
        </div>
      ) : (
        <p className="text-muted-foreground text-xs">
          {t("Save this AS2 profile to get the inbound URL to give the partner.")}
        </p>
      )}

      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Rotate the inbound URL?")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "The current URL stops working immediately. The partner's messages are refused until they send to the new one.",
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={rotateMutation.isPending}
              onClick={() => profile && rotateMutation.mutate(profile.id)}
            >
              {t("Rotate")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </FormSection>
  );
}
