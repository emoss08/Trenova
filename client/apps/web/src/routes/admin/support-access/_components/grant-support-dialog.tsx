import { InputField } from "@/components/fields/input-field";
import { SegmentedField } from "@/components/fields/segmented-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  grantSupportAccess,
  type SupportAccessScope,
  type SupportAccessSettings,
} from "@/lib/graphql/support-access";
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
import { durationLabel, scopeLabel } from "./support-access-labels";

const grantSchema = z.object({
  staffEmail: z.string().trim().min(1).max(254),
  reason: z.string().trim().min(1).max(500),
  durationHours: z.number().int(),
  scope: z.enum(["ReadOnly", "Full"]),
});

type GrantValues = z.infer<typeof grantSchema>;

const defaults = (): GrantValues => ({
  staffEmail: "",
  reason: "",
  durationHours: 4,
  scope: "ReadOnly",
});

const SCOPES: readonly SupportAccessScope[] = ["ReadOnly", "Full"];

/**
 * Lets one named Trenova staff member into this organization for a set time. The server holds
 * every rule (the staff domain, the fixed durations, one live grant per person); this form only
 * offers what it will accept.
 */
export function GrantSupportDialog({
  open,
  settings,
  onClose,
  onGranted,
}: {
  open: boolean;
  settings: SupportAccessSettings;
  onClose: () => void;
  onGranted: () => Promise<void> | void;
}) {
  const t = useT();
  const form = useForm<GrantValues>({
    resolver: zodResolver(grantSchema),
    defaultValues: defaults(),
  });

  useEffect(() => {
    if (open) {
      form.reset(defaults());
    }
  }, [open, form]);

  const mutation = useApiMutation({
    mutationFn: (values: GrantValues) => grantSupportAccess(values),
    onSuccess: async () => {
      onClose();
      await onGranted();
    },
    form,
    resourceName: "Support access",
  });

  const submit = form.handleSubmit((values) => mutation.mutateAsync(values));
  const domain = settings.staffDomain ?? "";

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent size="md">
        <form onSubmit={submit}>
          <DialogHeader>
            <DialogTitle>{t("Grant Trenova support access")}</DialogTitle>
            <DialogDescription>
              {t(
                "One named Trenova staff member may enter this organization until the access ends. Everything they do is in the audit log, and you can end it at any time.",
              )}
            </DialogDescription>
          </DialogHeader>

          <FormGroup className="py-4">
            <FormControl>
              <InputField
                control={form.control}
                name="staffEmail"
                label={t("Staff member's email")}
                placeholder={domain === "" ? "" : t("name@{0}", domain)}
                autoComplete="off"
                required
                description={
                  domain === "" ? undefined : t("A Trenova staff address at {0}.", domain)
                }
              />
            </FormControl>
            <FormControl>
              <TextareaField
                control={form.control}
                name="reason"
                label={t("Reason")}
                placeholder={t("The ticket or problem support is looking into")}
                required
              />
            </FormControl>
            <FormControl>
              <SegmentedField
                control={form.control}
                name="durationHours"
                label={t("Ends after")}
                options={settings.durations.map((hours) => ({
                  value: hours,
                  label: durationLabel(t, hours),
                }))}
              />
            </FormControl>
            <FormControl>
              <SegmentedField
                control={form.control}
                name="scope"
                label={t("Access")}
                options={SCOPES.map((scope) => ({ value: scope, label: scopeLabel(t, scope) }))}
                description={t(
                  "Full access never includes managing users, roles or sign-in, API keys, stored credentials, bank or remittance details, or exporting data.",
                )}
              />
            </FormControl>
          </FormGroup>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={mutation.isPending} loadingText={t("Granting…")}>
              {t("Grant access")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
