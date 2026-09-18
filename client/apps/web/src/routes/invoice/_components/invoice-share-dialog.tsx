import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { absoluteAppUrl, invoicePanelPath } from "@/lib/invoice-links";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Invoice } from "@trenova/shared/types/invoice";
import {
  MAX_INVOICE_SHARE_NOTE_LENGTH,
  shareInvoiceFormSchema,
  type InvoiceDetailTab,
  type InvoiceShare,
  type InvoiceShareUser,
  type ShareInvoiceFormValues,
  type ShareInvoiceResult,
} from "@trenova/shared/types/invoice-share";
import { CheckIcon, CopyIcon, LinkIcon, Share2Icon, XIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useId, useState } from "react";
import { FormProvider, useController, useForm, type Control, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { invoiceDetailTabSearchParamsParser } from "../use-invoice-state";

const SHARE_FORM_ID = "invoice-share-form";
const COPY_FEEDBACK_MS = 3000;
const CANDIDATE_SEARCH_DEBOUNCE_MS = 200;
const EMPTY_FORM: ShareInvoiceFormValues = { userIds: [], note: "" };

const SECTION_CARD = "flex min-w-0 flex-col gap-5 rounded-xl border bg-background p-5";
const FIELD = "h-9 w-full min-w-0 rounded-lg bg-background text-sm";

export function InvoiceShareDialog({ invoice }: { invoice: Invoice }) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
        <Share2Icon className="size-3.5" />
        {t("Share")}
      </Button>
      {open ? <InvoiceShareDialogContent invoice={invoice} /> : null}
    </Dialog>
  );
}

function InvoiceShareDialogContent({ invoice }: { invoice: Invoice }) {
  const t = useT();
  const [{ tab }] = useQueryStates(invoiceDetailTabSearchParamsParser);
  const link = absoluteAppUrl(invoicePanelPath(invoice.id, tab));
  const { copy, isCopied } = useCopyToClipboard();

  const copyLink = () => {
    void copy(link, { timeout: COPY_FEEDBACK_MS, withToast: true });
  };

  return (
    <DialogContent
      showCloseButton={false}
      className="max-h-[calc(100dvh-2rem)] gap-6 overflow-y-auto p-5 sm:max-w-[560px]"
    >
      <section className={SECTION_CARD}>
        <div className="relative flex flex-col gap-1 pr-8">
          <DialogTitle className="text-lg font-semibold">
            {t("Share invoice {0}", invoice.number)}
          </DialogTitle>
          <DialogDescription>
            {t("Send teammates a link to this invoice. Sharing doesn't change who can see it.")}
          </DialogDescription>
          <DialogClose
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                className="absolute -top-1 -right-1"
                aria-label={t("Close")}
              >
                <XIcon className="size-4" />
              </Button>
            }
          />
        </div>

        <div className="bg-muted/40 flex min-w-0 flex-col gap-4 rounded-xl border p-4">
          <div className="flex items-start justify-between gap-4">
            <div className="flex min-w-0 flex-col gap-0.5">
              <h3 className="text-sm font-semibold">{t("Direct link")}</h3>
              <p className="text-muted-foreground text-sm">
                {t("Anyone whose role lets them view invoices can open it.")}
              </p>
            </div>
            <AccessLabel />
          </div>
          <Input
            readOnly
            value={link}
            aria-label={t("Invoice link")}
            onFocus={(event) => event.currentTarget.select()}
            inputContainerClassName="w-full"
            className={cn(FIELD, "pr-20 pl-9 truncate")}
            leftElement={<LinkIcon className="text-muted-foreground size-4" />}
            rightElement={
              <button
                type="button"
                onClick={copyLink}
 className="ui-focus-ring text-muted-foreground hover:text-foreground mr-1 flex h-7 items-center gap-1 rounded-md px-2 text-sm transition-colors"
              >
                {isCopied ? <CheckIcon className="size-3.5 text-success-foreground" /> : null}
                {isCopied ? t("Copied") : t("Copy")}
              </button>
            }
          />
        </div>
      </section>

      <section className={cn(SECTION_CARD)}>
        <div className="flex flex-col gap-0.5">
          <h3 className="text-lg font-semibold">{t("Invite teammates")}</h3>
          <p className="text-muted-foreground text-sm">
            {t("Add teammates by name, username, or email.")}
          </p>
        </div>
        <InviteForm invoice={invoice} tab={tab} />
        <SharedWithList invoiceId={invoice.id} />
      </section>

      <div className="flex items-center justify-between gap-3">
        <Button type="button" variant="outline" className="h-9 rounded-lg px-3" onClick={copyLink}>
          {isCopied ? <CheckIcon className="size-4" /> : <CopyIcon className="size-4" />}
          {isCopied ? t("Copied") : t("Copy link")}
        </Button>
        <DialogClose
          render={
            <Button
              type="button"
              className="bg-foreground text-background hover:bg-foreground/90 h-9 rounded-lg px-4"
            >
              {t("Done")}
            </Button>
          }
        />
      </div>
    </DialogContent>
  );
}

function AccessLabel() {
  const t = useT();

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span tabIndex={0} className="text-muted-foreground shrink-0 text-sm whitespace-nowrap">
            {t("Can view")}
          </span>
        }
      />
      <TooltipContent side="top" className="max-w-60">
        {t("What each person can do comes from their role. Sharing never grants access.")}
      </TooltipContent>
    </Tooltip>
  );
}

function InviteForm({ invoice, tab }: { invoice: Invoice; tab: InvoiceDetailTab }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<InvoiceShareUser | null>(null);
  const [inviteCount, setInviteCount] = useState(0);

  const form = useForm<ShareInvoiceFormValues>({
    resolver: zodResolver(shareInvoiceFormSchema) as Resolver<ShareInvoiceFormValues>,
    defaultValues: EMPTY_FORM,
  });
  const { control, handleSubmit, reset, register, formState } = form;

  const mutation = useApiMutation({
    mutationFn: (values: ShareInvoiceFormValues) =>
      apiService.invoiceShareService.share(invoice.id, { ...values, tab }),
    onSuccess: (result) => {
      queryClient.setQueryData(queries["invoice-share"].list(invoice.id).queryKey, result.shares);
      toast.success(shareSuccessMessage(t, result));
      setSelected(null);
      setInviteCount((count) => count + 1);
      reset(EMPTY_FORM);
    },
    form,
    resourceName: "Invoice Share",
  });

  const onSubmit = handleSubmit((values) => mutation.mutate(values));
  const noteError = formState.errors.note?.message;

  return (
    <FormProvider {...form}>
      <Form id={SHARE_FORM_ID} onSubmit={onSubmit} className="flex flex-col gap-3">
        <div className="flex items-start gap-3">
          <TeammateSearch
            key={inviteCount}
            invoiceId={invoice.id}
            control={control}
            selected={selected}
            onSelectedChange={setSelected}
          />
          <Button
            type="submit"
            form={SHARE_FORM_ID}
            variant="outline"
            className="h-9 shrink-0 rounded-lg px-4"
            isLoading={mutation.isPending}
            loadingText={t("Inviting...")}
          >
            {t("Invite")}
          </Button>
        </div>
        {selected ? (
          <div className="flex flex-col gap-1">
            <Textarea
              {...register("note")}
              aria-label={t("Note")}
              placeholder={t("Add a note for {0} (optional)", selected.name)}
              maxLength={MAX_INVOICE_SHARE_NOTE_LENGTH}
              minRows={4}
              className="bg-background rounded-lg text-sm"
              aria-invalid={noteError ? true : undefined}
            />
            {noteError ? (
              <p className="text-destructive text-xs">{t(noteError)}</p>
            ) : (
              <p className="text-muted-foreground text-xs">
                {t("Anyone who can view this invoice sees the note in the list below.")}
              </p>
            )}
          </div>
        ) : null}
      </Form>
    </FormProvider>
  );
}

function TeammateSearch({
  invoiceId,
  control,
  selected,
  onSelectedChange,
}: {
  invoiceId: string;
  control: Control<ShareInvoiceFormValues>;
  selected: InvoiceShareUser | null;
  onSelectedChange: (user: InvoiceShareUser | null) => void;
}) {
  const t = useT();
  const listboxId = useId();
  const { field, fieldState } = useController({ control, name: "userIds" });
  const [text, setText] = useState("");
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const query = useDebounce(text, CANDIDATE_SEARCH_DEBOUNCE_MS);

  const showList = open && !selected;
  const { data: candidates = [], isFetching } = useQuery({
    ...queries["invoice-share"].candidates(invoiceId, query),
    enabled: showList,
  });
  const activeOption = showList ? candidates[activeIndex] : undefined;

  const choose = (candidate: InvoiceShareUser) => {
    onSelectedChange(candidate);
    field.onChange([candidate.id]);
    setText(candidate.name);
    setOpen(false);
  };

  const clear = () => {
    onSelectedChange(null);
    field.onChange([]);
    setText("");
    setOpen(true);
  };

  const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "ArrowDown") {
      event.preventDefault();
      setOpen(true);
      setActiveIndex((index) => Math.min(index + 1, Math.max(candidates.length - 1, 0)));
    } else if (event.key === "ArrowUp") {
      event.preventDefault();
      setActiveIndex((index) => Math.max(index - 1, 0));
    } else if (event.key === "Enter" && activeOption) {
      event.preventDefault();
      choose(activeOption);
    } else if (event.key === "Escape" && showList) {
      event.preventDefault();
      event.stopPropagation();
      setOpen(false);
    }
  };

  return (
    <div className="relative flex min-w-0 flex-1 flex-col gap-1">
      <Input
        role="combobox"
        aria-label={t("Teammate")}
        aria-expanded={showList}
        aria-controls={listboxId}
        aria-autocomplete="list"
        aria-activedescendant={activeOption ? `${listboxId}-${activeOption.id}` : undefined}
        aria-invalid={fieldState.error ? true : undefined}
        autoComplete="off"
        value={text}
        placeholder={t("Name, username, or email")}
        className={cn(FIELD, selected && "pr-9")}
        inputContainerClassName="w-full"
        onChange={(event) => {
          if (selected) {
            onSelectedChange(null);
            field.onChange([]);
          }
          setText(event.target.value);
          setActiveIndex(0);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        onBlur={() => {
          field.onBlur();
          setOpen(false);
        }}
        onKeyDown={onKeyDown}
        rightElement={
          selected ? (
            <button
              type="button"
              onMouseDown={(event) => event.preventDefault()}
              onClick={clear}
              aria-label={t("Clear teammate")}
              className="text-muted-foreground hover:text-foreground mr-1 flex size-7 items-center justify-center rounded-md"
            >
              <XIcon className="size-3.5" />
            </button>
          ) : undefined
        }
      />
      {fieldState.error?.message ? (
        <p className="text-destructive text-xs">{t(fieldState.error.message)}</p>
      ) : null}
      {showList ? (
        <ul
          id={listboxId}
          role="listbox"
          aria-label={t("Teammates who can view invoices")}
          className="bg-popover text-popover-foreground absolute top-10 right-0 left-0 z-50 max-h-60 overflow-y-auto rounded-lg border p-1"
        >
          {candidates.length === 0 ? (
            <li className="text-muted-foreground px-2 py-2 text-sm">
              {isFetching ? t("Searching...") : t("No teammates who can view invoices match.")}
            </li>
          ) : (
            candidates.map((candidate, index) => (
              <li
                key={candidate.id}
                id={`${listboxId}-${candidate.id}`}
                role="option"
                aria-selected={index === activeIndex}
                onMouseDown={(event) => event.preventDefault()}
                onMouseEnter={() => setActiveIndex(index)}
                onClick={() => choose(candidate)}
                className={cn(
                  "flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5",
                  index === activeIndex && "bg-accent text-accent-foreground",
                )}
              >
                <PersonAvatar user={candidate} />
                <span className="truncate text-sm font-medium">{candidate.name}</span>
                {candidate.emailAddress ? (
                  <span className="text-muted-foreground truncate text-xs">
                    {candidate.emailAddress}
                  </span>
                ) : null}
              </li>
            ))
          )}
        </ul>
      ) : null}
    </div>
  );
}

function shareSuccessMessage(t: TranslateFn, result: ShareInvoiceResult): string {
  switch (result.emailStatus) {
    case "Queued":
      return t(
        "Shared with {0, plural, one {# teammate} other {# teammates}}. They'll get a notification and an email.",
        result.recipientCount,
      );
    case "Partial":
      return t(
        "Shared with {0, plural, one {# teammate} other {# teammates}}. Everyone got a notification, but some emails couldn't be sent.",
        result.recipientCount,
      );
    case "Failed":
      return t(
        "Shared with {0, plural, one {# teammate} other {# teammates}}. Everyone got a notification, but the emails couldn't be sent.",
        result.recipientCount,
      );
    case "NotConfigured":
      return t(
        "Shared with {0, plural, one {# teammate} other {# teammates}}. They'll see it in their notifications; email isn't set up for shares.",
        result.recipientCount,
      );
  }
}

function PersonAvatar({ user }: { user: InvoiceShareUser }) {
  return (
    <ResolvedUserAvatar
      userId={user.id}
      name={user.name}
      profilePicUrl={user.profilePicUrl}
      thumbnailUrl={user.thumbnailUrl}
      size="sm"
      className="rounded-full after:rounded-full"
      imageClassName="rounded-full"
      fallbackClassName="rounded-full text-2xs"
    />
  );
}

function SharedWithList({ invoiceId }: { invoiceId: string }) {
  const t = useT();
  const { data: shares, isLoading, isError } = useQuery(queries["invoice-share"].list(invoiceId));

  if (isLoading) {
    return (
      <div className="flex flex-col gap-3" aria-busy="true">
        <Skeleton className="h-6 w-full" />
        <Skeleton className="h-6 w-full" />
      </div>
    );
  }

  if (isError) {
    return (
      <p className="text-destructive text-sm">
        {t("Couldn't load who this invoice is shared with.")}
      </p>
    );
  }

  if (!shares || shares.length === 0) {
    return <p className="text-muted-foreground text-sm">{t("Not shared with anyone yet.")}</p>;
  }

  return (
    <ul className="flex max-h-64 flex-col gap-4 overflow-y-auto" aria-label={t("Shared with")}>
      {shares.map((share) => (
        <SharedWithRow key={share.id} share={share} />
      ))}
    </ul>
  );
}

function SharedWithRow({ share }: { share: InvoiceShare }) {
  const t = useT();
  const recipient = share.sharedWith;
  const name = recipient?.name || recipient?.emailAddress || t("Unknown user");
  const sharedBy = share.sharedBy?.name || t("Unknown user");

  return (
    <li className="flex min-w-0 items-center gap-3">
      {recipient ? <PersonAvatar user={recipient} /> : null}
      <Tooltip>
        <TooltipTrigger
          render={
            <div className="flex min-w-0 flex-1 items-baseline gap-2" tabIndex={0}>
              <span className="truncate text-sm font-medium">{name}</span>
              {recipient?.emailAddress ? (
                <span className="text-muted-foreground truncate text-sm">
                  {recipient.emailAddress}
                </span>
              ) : null}
            </div>
          }
        />
        <TooltipContent side="top" className="max-w-72">
          <p>{t("Shared by {0} on {1}", sharedBy, formatUnixDateTime(share.lastSharedAt))}</p>
          {share.note ? <p className="mt-1 whitespace-pre-line">{share.note}</p> : null}
        </TooltipContent>
      </Tooltip>
      <AccessLabel />
    </li>
  );
}
