import { CustomerAutocompleteField } from "@/components/autocomplete-fields";
import { FactoringNoaStatusBadge } from "@/components/factoring/factoring-badges";
import {
  LazyAcknowledgeFactoringNoaDialog,
  LazyReleaseFactoringNoaDialog,
  LazySendFactoringNoaDialog,
  preloadFactoringNoaDialogs,
} from "@/components/factoring/factoring-noa-dialogs-lazy";
import { TextareaField } from "@/components/fields/textarea-field";
import { SectionPanel } from "@/components/section-panel";
import { FactoringNoticesBodySkeleton } from "./factoring-notices-section-skeleton";
import { recordPath } from "@/config/record-links";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { draftFactoringNoa, type FactoringNoa } from "@/lib/graphql/factoring-noa";
import { factoringNoa as factoringNoaQueries } from "@/lib/queries/factoring-noa";
import { invalidateFactoringQueries } from "@/lib/queries/factoring-roots";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  FileCheck02Icon,
  PlusIcon,
  Send01Icon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { dateToUnixTimestamp, formatUnixDate } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { Link } from "react-router";
import { toast } from "sonner";
import { z } from "zod";

/**
 * Every notice of assignment this factor has with a customer: drafted, sent
 * and waiting on a signature, acknowledged (which is what lets that
 * customer's invoices be factored), or released.
 */
export function FactoringNoticesSection({
  factoringCompanyId,
  highlightedId,
}: {
  factoringCompanyId: string;
  highlightedId?: string | null;
}) {
  const t = useT();
  const { allowed: canRead } = usePermission(Resource.FactoringNoa, Operation.Read);
  const { allowed: canCreate } = usePermission(Resource.FactoringNoa, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.FactoringNoa, Operation.Update);
  const { data, isLoading, isError, refetch } = useQuery({
    ...factoringNoaQueries.list({ factoringCompanyId }),
    enabled: canRead,
  });
  const [nowUnix] = useState(() => dateToUnixTimestamp(new Date()));
  const [drafting, setDrafting] = useState(false);
  const [sending, setSending] = useState<FactoringNoa | null>(null);
  const [acknowledging, setAcknowledging] = useState<FactoringNoa | null>(null);
  const [releasing, setReleasing] = useState<FactoringNoa | null>(null);

  if (!canRead) {
    return null;
  }

  const notices = data ?? [];

  return (
    <SectionPanel
      title={t("Notices of assignment")}
      count={notices.length}
      help={t(
        "A customer's invoices can only be factored once it has acknowledged a notice telling it to pay this factor.",
      )}
      action={
        canCreate ? (
          <Button
            type="button"
            size="sm"
            onPointerEnter={preloadFactoringNoaDialogs}
            onFocus={preloadFactoringNoaDialogs}
            onClick={() => setDrafting(true)}
          >
            <PlusIcon />
            {t("New notice")}
          </Button>
        ) : null
      }
    >
      {isLoading ? (
        <FactoringNoticesBodySkeleton />
      ) : isError ? (
        <div className="p-3">
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("The notices could not be loaded.")}</span>
              <Button type="button" size="sm" variant="outline" onClick={() => void refetch()}>
                {t("Retry")}
              </Button>
            </AlertDescription>
          </Alert>
        </div>
      ) : notices.length === 0 ? (
        <p className="text-muted-foreground px-3 py-3 text-sm">
          {t("No notices yet. Draft one for each customer whose invoices you sell to this factor.")}
        </p>
      ) : (
        <ul className="divide-y">
          {notices.map((noa) => (
            <NoticeRow
              key={noa.id}
              noa={noa}
              nowUnix={nowUnix}
              highlighted={noa.id === highlightedId}
              canUpdate={canUpdate}
              onSend={() => setSending(noa)}
              onAcknowledge={() => setAcknowledging(noa)}
              onRelease={() => setReleasing(noa)}
            />
          ))}
        </ul>
      )}
      <DraftNoticeDialog
        factoringCompanyId={factoringCompanyId}
        open={drafting}
        onOpenChange={setDrafting}
        onDrafted={(noa) => {
          if (canUpdate) setSending(noa);
        }}
      />
      <LazySendFactoringNoaDialog
        noa={sending}
        onOpenChange={(open) => {
          if (!open) setSending(null);
        }}
      />
      <LazyAcknowledgeFactoringNoaDialog
        noa={acknowledging}
        onOpenChange={(open) => {
          if (!open) setAcknowledging(null);
        }}
      />
      <LazyReleaseFactoringNoaDialog
        noa={releasing}
        onOpenChange={(open) => {
          if (!open) setReleasing(null);
        }}
      />
    </SectionPanel>
  );
}

function NoticeRow({
  noa,
  nowUnix,
  highlighted,
  canUpdate,
  onSend,
  onAcknowledge,
  onRelease,
}: {
  noa: FactoringNoa;
  nowUnix: number;
  highlighted: boolean;
  canUpdate: boolean;
  onSend: () => void;
  onAcknowledge: () => void;
  onRelease: () => void;
}) {
  const t = useT();
  const open = noa.status === "Draft" || noa.status === "Sent";

  return (
    <li
      className={cn("flex flex-col gap-2 px-3 py-3", highlighted && "bg-brand-subtle")}
      data-testid="factoring-noa-row"
      aria-current={highlighted ? "true" : undefined}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <FactoringNoaStatusBadge noa={noa} nowUnix={nowUnix} />
          <Link
            to={recordPath("customer", noa.customerId)}
            className="truncate text-sm font-medium hover:underline"
          >
            {noa.customer?.name ?? noa.customerId}
          </Link>
        </div>
        {canUpdate && noa.status !== "Released" ? (
          <div className="flex flex-wrap items-center gap-2">
            {open ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onPointerEnter={preloadFactoringNoaDialogs}
                onFocus={preloadFactoringNoaDialogs}
                onClick={onSend}
              >
                <Send01Icon />
                {noa.status === "Sent" ? t("Send again") : t("Send")}
              </Button>
            ) : null}
            {open ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onPointerEnter={preloadFactoringNoaDialogs}
                onFocus={preloadFactoringNoaDialogs}
                onClick={onAcknowledge}
              >
                <FileCheck02Icon />
                {t("Record acknowledgement")}
              </Button>
            ) : null}
            <Button
              type="button"
              size="sm"
              variant="outline"
              onPointerEnter={preloadFactoringNoaDialogs}
              onFocus={preloadFactoringNoaDialogs}
              onClick={onRelease}
            >
              <SlashCircle01Icon />
              {t("Release")}
            </Button>
          </div>
        ) : null}
      </div>
      <DescriptionList columns={4}>
        <DescriptionItem label={t("Sent on")}>
          {noa.sentAt ? formatUnixDate(noa.sentAt) : <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Acknowledged on")}>
          {noa.acknowledgedAt ? formatUnixDate(noa.acknowledgedAt) : <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Expires")}>
          {noa.expiresAt ? formatUnixDate(noa.expiresAt) : <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Released on")}>
          {noa.releasedAt ? formatUnixDate(noa.releasedAt) : <DescriptionEmpty />}
        </DescriptionItem>
        {noa.sentTo ? (
          <DescriptionItem label={t("Sent to")} span="full">
            {noa.sentTo}
          </DescriptionItem>
        ) : null}
        {noa.releaseReason ? (
          <DescriptionItem label={t("Release reason")} span="full">
            {noa.releaseReason}
          </DescriptionItem>
        ) : null}
      </DescriptionList>
    </li>
  );
}

const MAX_NOA_NOTES = 2000;

const draftSchema = z.object({
  customerId: z.string().min(1, { error: () => translate("Customer is required") }),
  notes: z
    .string()
    .max(MAX_NOA_NOTES, { error: () => translate("Notes must be at most 2,000 characters") }),
});
type DraftValues = z.infer<typeof draftSchema>;

const DRAFT_FORM_ID = "draft-factoring-noa-form";

function DraftNoticeDialog({
  factoringCompanyId,
  open,
  onOpenChange,
  onDrafted,
}: {
  factoringCompanyId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDrafted: (noa: FactoringNoa) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<DraftValues>({
    resolver: zodResolver(draftSchema) as Resolver<DraftValues>,
    defaultValues: { customerId: "", notes: "" },
  });
  const { reset } = form;

  useEffect(() => {
    if (open) reset({ customerId: "", notes: "" });
  }, [open, reset]);

  const mutation = useApiMutation<FactoringNoa, DraftValues, unknown, DraftValues>({
    form,
    resourceName: "notice of assignment",
    mutationFn: (values) =>
      draftFactoringNoa({
        customerId: values.customerId,
        factoringCompanyId,
        notes: values.notes.trim() === "" ? null : values.notes.trim(),
      }),
    onSuccess: (noa) => {
      invalidateFactoringQueries(queryClient);
      toast.success(t("Notice drafted"));
      onOpenChange(false);
      onDrafted(noa);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t("New notice")}</DialogTitle>
          <DialogDescription>
            {t(
              "Drafts the letter telling a customer to pay this factor. Review it, then send it for the customer to sign.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            id={DRAFT_FORM_ID}
            onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
          >
            <FormGroup cols={1}>
              <FormControl>
                <CustomerAutocompleteField
                  control={form.control}
                  name="customerId"
                  label={t("Customer")}
                  placeholder={t("Select a customer")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <TextareaField control={form.control} name="notes" label={t("Notes")} rows={2} />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form={DRAFT_FORM_ID} isLoading={mutation.isPending}>
            {t("Save draft")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
