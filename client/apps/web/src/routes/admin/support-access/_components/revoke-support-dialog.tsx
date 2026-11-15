import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { revokeSupportAccess, type SupportAccessGrant } from "@/lib/graphql/support-access";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const revokeSchema = z.object({ reason: z.string().trim().min(1).max(500) });

type RevokeValues = z.infer<typeof revokeSchema>;

/** Ends a grant and every session under it at once, with the reason the audit log keeps. */
export function RevokeSupportDialog({
  grant,
  onClose,
  onRevoked,
}: {
  grant: SupportAccessGrant | null;
  onClose: () => void;
  onRevoked: () => Promise<void> | void;
}) {
  const t = useT();
  const form = useForm<RevokeValues>({
    resolver: zodResolver(revokeSchema),
    defaultValues: { reason: "" },
  });

  useEffect(() => {
    if (grant !== null) {
      form.reset({ reason: "" });
    }
  }, [grant, form]);

  const mutation = useApiMutation({
    mutationFn: (values: RevokeValues) => {
      if (grant === null) {
        throw new Error("no grant to revoke");
      }
      return revokeSupportAccess(grant.id, values.reason);
    },
    onSuccess: async () => {
      onClose();
      await onRevoked();
    },
    form,
    resourceName: "Support access",
  });

  const submit = form.handleSubmit((values) => mutation.mutateAsync(values));

  return (
    <Dialog open={grant !== null} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="sm">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>{t("End Trenova support access?")}</DialogTitle>
            <DialogDescription>
              {grant === null
                ? null
                : t(
                    "{0} loses access now, and any session they have open ends at once.",
                    grant.staffEmail,
                  )}
            </DialogDescription>
          </DialogHeader>
          <FormGroup className="py-4">
            <FormControl>
              <TextareaField
                control={form.control}
                name="reason"
                label={t("Reason")}
                placeholder={t("Resolved, or no longer needed")}
                required
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t("Cancel")}
            </Button>
            <Button
              type="submit"
              variant="destructive"
              isLoading={mutation.isPending}
              loadingText={t("Ending…")}
            >
              {t("End access")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
