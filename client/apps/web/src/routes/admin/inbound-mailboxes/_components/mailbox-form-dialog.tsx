import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createInboundMailbox,
  setInboundMailboxSigningSecret,
  updateInboundMailbox,
  type InboundMailbox,
  type InboundMailboxCredentials,
} from "@/lib/graphql/inbox";
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
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import {
  MAILBOX_STATUSES,
  PROVIDERS,
  REVIEW_POLICIES,
  mailboxFormSchema,
  mailboxFormValues,
  mailboxInput,
  newMailboxDefaults,
  type MailboxFormValues,
} from "./mailbox-schema";
import { generatePostmarkCredentials, postmarkSecret, type BasicCredentials } from "./webhook-url";

export function reviewPolicyLabel(t: (value: string) => string, policy: string): string {
  switch (policy) {
    case "AlwaysReview":
      return t("A person reviews everything");
    case "ReviewBelowConfidence":
      return t("The desk acts when it is sure enough");
    case "AutoHandle":
      return t("The desk acts on everything it can");
    default:
      return policy;
  }
}

export type MailboxCreated = {
  credentials: InboundMailboxCredentials;
  postmark?: BasicCredentials;
};

/**
 * Creates or edits a mailbox. Creating one is the whole setup: a Resend
 * mailbox can take its signing secret here, and a Postmark one is given
 * generated credentials and saved with them, so the next screen's URL is
 * everything the provider needs.
 */
export function MailboxFormDialog({
  open,
  mailbox,
  onClose,
  onCreated,
  onSaved,
}: {
  open: boolean;
  mailbox: InboundMailbox | null;
  onClose: () => void;
  onCreated: (created: MailboxCreated) => void;
  onSaved: () => Promise<void> | void;
}) {
  const t = useT();
  const editing = mailbox !== null;
  const [resendSecret, setResendSecret] = useState("");

  const form = useForm<MailboxFormValues>({
    resolver: zodResolver(mailboxFormSchema),
    defaultValues: mailbox === null ? newMailboxDefaults() : mailboxFormValues(mailbox),
  });

  useEffect(() => {
    if (open) {
      form.reset(mailbox === null ? newMailboxDefaults() : mailboxFormValues(mailbox));
    }
  }, [open, mailbox, form]);

  // The secret is typed into a plain input, not the form, so it is cleared
  // here on every way out rather than left for the next mailbox to inherit.
  const close = () => {
    setResendSecret("");
    onClose();
  };

  const provider = useWatch({ control: form.control, name: "provider" });
  const reviewPolicy = useWatch({ control: form.control, name: "reviewPolicy" });

  const createMutation = useApiMutation({
    mutationFn: async (values: MailboxFormValues): Promise<MailboxCreated> => {
      const secret = values.provider === "Resend" ? resendSecret.trim() : "";
      const credentials = await createInboundMailbox(
        mailboxInput(values),
        secret === "" ? null : secret,
      );
      if (values.provider !== "Postmark") {
        return { credentials };
      }

      // Postmark's credentials are ours to choose, so the mailbox is born
      // with them rather than left refusing mail until somebody comes back.
      const postmark = generatePostmarkCredentials();
      const updated = await setInboundMailboxSigningSecret(
        credentials.mailbox.id,
        postmarkSecret(postmark),
      );

      return { credentials: { ...credentials, mailbox: updated }, postmark };
    },
    onSuccess: async (created) => {
      toast.success(t("Mailbox created"));
      setResendSecret("");
      onCreated(created);
      await onSaved();
    },
    form,
    resourceName: "Mailbox",
  });

  const updateMutation = useApiMutation({
    mutationFn: (values: MailboxFormValues) => {
      if (mailbox === null) {
        throw new Error("no mailbox to update");
      }
      return updateInboundMailbox(mailbox.id, mailbox.version, mailboxInput(values));
    },
    onSuccess: async (updated) => {
      toast.success(
        mailbox !== null && updated.provider !== mailbox.provider
          ? t("Saved. Set the new provider's signing secret before mail will be accepted.")
          : t("Mailbox saved"),
      );
      close();
      await onSaved();
    },
    form,
    resourceName: "Mailbox",
  });

  const busy = createMutation.isPending || updateMutation.isPending;
  const submit = form.handleSubmit((values) =>
    editing ? updateMutation.mutate(values) : createMutation.mutate(values),
  );

  return (
    <Dialog open={open} onOpenChange={(next) => !next && close()}>
      <DialogContent size="lg">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>{editing ? t("Edit {0}", mailbox.name) : t("New mailbox")}</DialogTitle>
            <DialogDescription>
              {t(
                "An address the system listens on. Mail a provider forwards here lands in the inbox, read and matched.",
              )}
            </DialogDescription>
          </DialogHeader>

          <FormGroup cols={2} className="py-4">
            <FormControl>
              <InputField
                control={form.control}
                name="name"
                label={t("Name")}
                placeholder={t("Tenders")}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={form.control}
                name="address"
                label={t("Address")}
                placeholder="tenders@yourcompany.com"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={form.control}
                name="provider"
                label={t("Provider")}
                options={PROVIDERS.map((value) => ({ value, label: value }))}
                description={
                  editing
                    ? t(
                        "Changing the provider clears the signing secret, which belongs to the old one.",
                      )
                    : undefined
                }
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={form.control}
                name="status"
                label={t("Status")}
                options={MAILBOX_STATUSES.map((value) => ({
                  value,
                  label: value === "Active" ? t("Listening") : t("Not listening"),
                }))}
              />
            </FormControl>
            <FormControl cols="full">
              <InputField
                control={form.control}
                name="purpose"
                label={t("Purpose")}
                placeholder={t("Tenders, rate confirmations and PODs from shippers")}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={form.control}
                name="reviewPolicy"
                label={t("How much it may do alone")}
                options={REVIEW_POLICIES.map((value) => ({
                  value,
                  label: reviewPolicyLabel(t, value),
                }))}
                description={t(
                  "Nothing that creates a load or moves money is ever done without a person.",
                )}
              />
            </FormControl>
            {reviewPolicy === "ReviewBelowConfidence" && (
              <FormControl>
                <NumberField
                  control={form.control}
                  name="minConfidence"
                  label={t("Confidence bar")}
                  description={t("Between 0.5 and 1. Below the bar a message waits for a person.")}
                />
              </FormControl>
            )}
            {!editing && provider === "Resend" && (
              <FormControl cols="full">
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="new-mailbox-secret">{t("Signing secret (optional)")}</Label>
                  <Input
                    id="new-mailbox-secret"
                    type="password"
                    autoComplete="off"
                    value={resendSecret}
                    onChange={(event) => setResendSecret(event.target.value)}
                    placeholder="whsec_…"
                  />
                  <p className="text-foreground-subtle text-xs">
                    {t(
                      "From the Resend webhook. You can add it later; until then the mailbox refuses deliveries.",
                    )}
                  </p>
                </div>
              </FormControl>
            )}
          </FormGroup>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={close}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={busy} loadingText={t("Saving…")}>
              {editing ? t("Save changes") : t("Create mailbox")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
