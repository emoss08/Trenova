import { SegmentedField } from "@/components/fields/segmented-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { extendSupportAccess, type SupportAccessGrant } from "@/lib/graphql/support-access";
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
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { durationLabel } from "./support-access-labels";

const extendSchema = z.object({ durationHours: z.number().int() });

type ExtendValues = z.infer<typeof extendSchema>;

/** Lets a live grant last longer. The new length counts from now and never shortens access. */
export function ExtendSupportDialog({
  grant,
  durations,
  onClose,
  onExtended,
}: {
  grant: SupportAccessGrant | null;
  durations: number[];
  onClose: () => void;
  onExtended: () => Promise<void> | void;
}) {
  const t = useT();
  const form = useForm<ExtendValues>({
    resolver: zodResolver(extendSchema),
    defaultValues: { durationHours: 4 },
  });

  useEffect(() => {
    if (grant !== null) {
      form.reset({ durationHours: 4 });
    }
  }, [grant, form]);

  const mutation = useApiMutation({
    mutationFn: (values: ExtendValues) => {
      if (grant === null) {
        throw new Error("no grant to extend");
      }
      return extendSupportAccess(grant.id, values.durationHours);
    },
    onSuccess: async () => {
      onClose();
      await onExtended();
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
            <DialogTitle>{t("Extend Trenova support access")}</DialogTitle>
            <DialogDescription>
              {grant === null
                ? null
                : t(
                    "{0} has access until {1}. The new length counts from now.",
                    grant.staffEmail,
                    formatUnixDateTime(grant.expiresAt),
                  )}
            </DialogDescription>
          </DialogHeader>
          <FormGroup className="py-4">
            <FormControl>
              <SegmentedField
                control={form.control}
                name="durationHours"
                label={t("Ends after")}
                options={durations.map((hours) => ({
                  value: hours,
                  label: durationLabel(t, hours),
                }))}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              {t("Cancel")}
            </Button>
            <Button type="submit" isLoading={mutation.isPending} loadingText={t("Extending…")}>
              {t("Extend access")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
