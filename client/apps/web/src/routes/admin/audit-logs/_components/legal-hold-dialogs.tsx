import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  AUDIT_LEGAL_HOLD_LIST_KEY,
  placeAuditLegalHold,
  releaseAuditLegalHold,
} from "@/lib/graphql/audit-log-admin";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { z } from "zod";

function placeSchema(t: TranslateFn) {
  return z
    .object({
      name: z.string().trim().min(1, t("Name the hold")).max(200),
      reason: z.string().trim().min(1, t("Say why the entries must be kept")).max(4000),
      resource: z.string().trim(),
      resourceId: z.string().trim(),
    })
    .refine((v) => v.resourceId === "" || v.resource !== "", {
      path: ["resource"],
      message: t("Name the kind of record the hold covers"),
    });
}

type PlaceValues = z.infer<ReturnType<typeof placeSchema>>;

const EMPTY_PLACE: PlaceValues = { name: "", reason: "", resource: "", resourceId: "" };

export function PlaceLegalHoldDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<PlaceValues>({
    resolver: zodResolver(placeSchema(t)) as Resolver<PlaceValues>,
    defaultValues: EMPTY_PLACE,
  });

  useEffect(() => {
    if (open) {
      form.reset(EMPTY_PLACE);
    }
  }, [open, form]);

  const place = useApiMutation({
    form,
    scope: { id: "audit-legal-hold-place" },
    resourceName: "Legal hold",
    mutationFn: (values: PlaceValues) =>
      placeAuditLegalHold({
        name: values.name,
        reason: values.reason,
        resource: values.resource || null,
        resourceId: values.resourceId || null,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [AUDIT_LEGAL_HOLD_LIST_KEY] });
      onOpenChange(false);
    },
  });

  const submit = form.handleSubmit((values) => place.mutateAsync(values).catch(() => undefined));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Place a legal hold")}</DialogTitle>
          <DialogDescription>
            {t(
              "Entries under a hold are kept past their retention until it is released. Leave the kind of record empty to hold every entry.",
            )}
          </DialogDescription>
        </DialogHeader>
        <form id="legal-hold-form" onSubmit={(event) => void submit(event)} className="grid gap-3">
          <InputField control={form.control} name="name" label={t("Name")} />
          <TextareaField control={form.control} name="reason" label={t("Reason")} />
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <InputField
              control={form.control}
              name="resource"
              label={t("Kind of record")}
              placeholder={t("For example shipment")}
            />
            <InputField
              control={form.control}
              name="resourceId"
              label={t("Record ID")}
              placeholder={t("Every record of the kind")}
            />
          </div>
        </form>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form="legal-hold-form" isLoading={place.isPending}>
            {t("Place hold")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function releaseSchema(t: TranslateFn) {
  return z.object({
    reason: z.string().trim().min(1, t("Say why the hold is no longer needed")).max(4000),
  });
}

type ReleaseValues = z.infer<ReturnType<typeof releaseSchema>>;

export function ReleaseLegalHoldDialog({
  hold,
  onOpenChange,
}: {
  hold: { id: string; name: string } | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<ReleaseValues>({
    resolver: zodResolver(releaseSchema(t)) as Resolver<ReleaseValues>,
    defaultValues: { reason: "" },
  });

  useEffect(() => {
    if (hold) {
      form.reset({ reason: "" });
    }
  }, [hold, form]);

  const release = useApiMutation({
    form,
    scope: { id: `audit-legal-hold-release:${hold?.id ?? ""}` },
    resourceName: "Legal hold",
    mutationFn: (values: ReleaseValues) => releaseAuditLegalHold(hold?.id ?? "", values.reason),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [AUDIT_LEGAL_HOLD_LIST_KEY] });
      onOpenChange(false);
    },
  });

  const submit = form.handleSubmit((values) => release.mutateAsync(values).catch(() => undefined));

  return (
    <Dialog open={hold !== null} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("Release {0}", hold?.name ?? "")}</DialogTitle>
          <DialogDescription>
            {t(
              "Retention may then delete the entries this hold kept. The release is recorded with your reason.",
            )}
          </DialogDescription>
        </DialogHeader>
        <form id="legal-hold-release-form" onSubmit={(event) => void submit(event)}>
          <TextareaField control={form.control} name="reason" label={t("Reason")} />
        </form>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="submit"
            form="legal-hold-release-form"
            variant="destructive"
            isLoading={release.isPending}
          >
            {t("Release hold")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
