import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { useRegister } from "@/hooks/use-register";
import { absoluteAppUrl } from "@/lib/app-url";
import {
  shareRecord,
  unshareRecord,
  type RecordShare,
  type RecordShareUser,
  type ShareRecordResult,
} from "@/lib/graphql/record-share";
import { recordShare } from "@/lib/queries/record-share";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormSection } from "@trenova/shared/components/ui/form";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent } from "@trenova/shared/components/ui/popover";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import {
  MAX_RECORD_SHARE_NOTE_LENGTH,
  shareRecordFormSchema,
  type ShareRecordFormValues,
} from "@trenova/shared/types/record-share";
import {
  CheckIcon,
  Copy01Icon,
  Link01Icon,
  Share07Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { useId, useRef, useState, type ReactNode } from "react";
import { FormProvider, useController, useForm, type Control, type Resolver } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";

const SHARE_FORM_ID = "record-share-form";
const COPY_FEEDBACK_MS = 3000;
const CANDIDATE_SEARCH_DEBOUNCE_MS = 200;
const EMPTY_FORM: ShareRecordFormValues = { userIds: [], note: "" };

const FIELD = "h-9 w-full min-w-0 rounded-lg bg-background text-sm";

/** A record that can be shared with teammates: its kind and id, and what it is called. */
export type SharedRecord = {
  /** The record kind, as the record-link registry names it: invoice, shipment, worker. */
  kind: string;
  id: string;
  /** The record in a sentence: its kind, then its label ("invoice INV-1042"). */
  title: string;
};

export type ShareDialogProps = {
  /**
   * The record being shared. Left out for a place that is no one record (a
   * conversation, a list): its link can be copied, and nobody can be invited
   * to it, because there is no record whose access says who may open it.
   */
  record?: SharedRecord | null;
  /** Where in the app the link opens: the page, tab and item the sharer is on. */
  path: string;
  /** The button that opens the dialog; the standard Share button when left out. */
  trigger?: (open: () => void) => ReactNode;
  /**
   * What sharing this does, in a sentence, where it differs from the plain
   * case of sending a link: a Desk case shared with someone comes into their
   * view to read.
   */
  description?: string;
};

/**
 * Share what is in front with teammates. One dialog for everything that can
 * be shared (an invoice, a case, an item in Needs you), over one resource: who
 * a record was shared with is kept and listed whenever the dialog is opened
 * for it again. A share sends a link in the app and by email; it never changes
 * who can see the record, so only teammates whose own role lets them open it
 * are offered.
 */
export function ShareDialog({ record = null, path, trigger, description }: ShareDialogProps) {
  const t = useT();
  const [open, setOpen] = useState(false);
  // Inviting teammates to a record takes the Share records permission. Without
  // it the dialog is the link alone, which anyone may copy: a link opens
  // nothing its reader could not already open.
  const { allowed: canShare } = usePermission(Resource.RecordShare, Operation.Create);
  const shared = canShare ? record : null;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {trigger ? (
        trigger(() => setOpen(true))
      ) : (
        <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
          <Share07Icon className="size-3.5" />
          {t("Share")}
        </Button>
      )}
      {open ? (
        <ShareDialogContent
          record={shared}
          path={path}
          description={shared ? description : undefined}
        />
      ) : null}
    </Dialog>
  );
}

function ShareDialogContent({
  record,
  path,
  description,
}: {
  record: SharedRecord | null;
  path: string;
  description?: string;
}) {
  const t = useT();
  const link = absoluteAppUrl(path);
  const { copy, isCopied } = useCopyToClipboard();

  const copyLink = () => {
    void copy(link, { timeout: COPY_FEEDBACK_MS, withToast: true });
  };

  return (
    <DialogContent size="md" className="max-h-[calc(100dvh-2rem)] overflow-y-auto">
      <DialogHeader>
        <DialogTitle>{record ? t("Share {0}", record.title) : t("Share this page")}</DialogTitle>
        <DialogDescription>
          {description ??
            (record
              ? t("Send teammates a link to this. Sharing doesn't change who can see it.")
              : t("Copy a link to this page. It opens for anyone who can already see it."))}
        </DialogDescription>
      </DialogHeader>
      <div className="flex min-w-0 flex-col gap-6">
        <FormSection
          title={t("Direct link")}
          description={t("Anyone whose role lets them view this can open it.")}
          action={<AccessLabel />}
        >
          <Input
            readOnly
            value={link}
            aria-label={t("Link")}
            onFocus={(event) => event.currentTarget.select()}
            inputContainerClassName="w-full"
            className={cn(FIELD, "truncate pr-20 pl-9")}
            leftElement={<Link01Icon className="size-4 text-muted-foreground" />}
            rightElement={
              <button
                type="button"
                onClick={copyLink}
                className="mr-1 flex h-7 items-center gap-1 rounded-md ui-focus-ring px-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
              >
                {isCopied ? <CheckIcon className="size-3.5 text-success-foreground" /> : null}
                {isCopied ? t("Copied") : t("Copy")}
              </button>
            }
          />
        </FormSection>
        {record ? (
          <FormSection
            title={t("Invite teammates")}
            description={t("Add teammates by name, username, or email.")}
          >
            <InviteForm record={record} path={path} />
            <SharedWithList record={record} />
          </FormSection>
        ) : null}
      </div>
      <DialogFooter className="sm:justify-between">
        <Button type="button" variant="ghost" onClick={copyLink}>
          {isCopied ? <CheckIcon className="size-4" /> : <Copy01Icon className="size-4" />}
          {isCopied ? t("Copied") : t("Copy link")}
        </Button>
        <DialogClose render={<Button type="button">{t("Done")}</Button>} />
      </DialogFooter>
    </DialogContent>
  );
}

function AccessLabel() {
  const t = useT();

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span tabIndex={0} className="shrink-0 text-sm whitespace-nowrap text-muted-foreground">
            {t("Can view")}
          </span>
        }
      />
      <TooltipContent side="top" className="max-w-60">
        {t(
          "What each person can do comes from their role. A share also lets them read this record's case until it is removed.",
        )}
      </TooltipContent>
    </Tooltip>
  );
}

function InviteForm({ record, path }: { record: SharedRecord; path: string }) {
  const t = useT();
  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<RecordShareUser | null>(null);
  const [inviteCount, setInviteCount] = useState(0);

  const form = useForm<ShareRecordFormValues>({
    resolver: zodResolver(shareRecordFormSchema) as Resolver<ShareRecordFormValues>,
    defaultValues: EMPTY_FORM,
  });
  const { control, handleSubmit, reset, formState } = form;

  const mutation = useApiMutation({
    mutationFn: (values: ShareRecordFormValues) =>
      shareRecord({ kind: record.kind, id: record.id, path, ...values }),
    onSuccess: (result) => {
      queryClient.setQueryData(recordShare.list(record.kind, record.id).queryKey, result.shares);
      toast.success(shareSuccessMessage(t, result));
      setSelected(null);
      setInviteCount((count) => count + 1);
      reset(EMPTY_FORM);
    },
    form,
    resourceName: "Share",
  });

  const onSubmit = handleSubmit((values) => mutation.mutate(values));
  const noteError = formState.errors.note?.message;

  return (
    <FormProvider {...form}>
      <Form id={SHARE_FORM_ID} onSubmit={onSubmit} className="flex flex-col gap-3">
        <div className="flex items-start gap-3">
          <TeammateSearch
            key={inviteCount}
            record={record}
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
        {selected ? <NoteField control={control} recipient={selected} error={noteError} /> : null}
      </Form>
    </FormProvider>
  );
}

/** The optional note for the picked teammate, registered only while it is drawn. */
function NoteField({
  control,
  recipient,
  error,
}: {
  control: Control<ShareRecordFormValues>;
  recipient: RecordShareUser;
  error?: string;
}) {
  const t = useT();
  const note = useRegister(control, "note");

  return (
    <div className="flex flex-col gap-1">
      <Textarea
        {...note}
        aria-label={t("Note")}
        placeholder={t("Add a note for {0} (optional)", recipient.name)}
        maxLength={MAX_RECORD_SHARE_NOTE_LENGTH}
        minRows={4}
        className="rounded-lg bg-background text-sm"
        aria-invalid={error ? true : undefined}
      />
      {error ? (
        <p className="text-xs text-destructive">{t(error)}</p>
      ) : (
        <p className="text-xs text-muted-foreground">
          {t("Anyone who can view this sees the note in the list below.")}
        </p>
      )}
    </div>
  );
}

function TeammateSearch({
  record,
  control,
  selected,
  onSelectedChange,
}: {
  record: SharedRecord;
  control: Control<ShareRecordFormValues>;
  selected: RecordShareUser | null;
  onSelectedChange: (user: RecordShareUser | null) => void;
}) {
  const t = useT();
  const listboxId = useId();
  const { field, fieldState } = useController({ control, name: "userIds" });
  const [text, setText] = useState("");
  const [open, setOpen] = useState(false);
  const [activeIndex, setActiveIndex] = useState(0);
  const query = useDebounce(text, CANDIDATE_SEARCH_DEBOUNCE_MS);
  const anchorRef = useRef<HTMLDivElement>(null);

  const showList = open && !selected;
  const { data: candidates = [], isFetching } = useQuery({
    ...recordShare.candidates(record.kind, record.id, query),
    enabled: showList,
  });
  const activeOption = showList ? candidates[activeIndex] : undefined;

  const choose = (candidate: RecordShareUser) => {
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
    <div ref={anchorRef} className="relative flex min-w-0 flex-1 flex-col gap-1">
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
              className="mr-1 flex size-7 items-center justify-center rounded-md text-muted-foreground hover:text-foreground"
            >
              <XCloseIcon className="size-3.5" />
            </button>
          ) : undefined
        }
      />
      {fieldState.error?.message ? (
        <p className="text-xs text-destructive">{t(fieldState.error.message)}</p>
      ) : null}
      {/* The list floats over the dialog rather than inside it, where the
          dialog's own scrolling would cut it off. Focus stays in the field. */}
      <Popover open={showList} modal={false}>
        <PopoverContent
          anchor={anchorRef}
          side="bottom"
          align="start"
          sideOffset={4}
          inverted={false}
          initialFocus={false}
          finalFocus={false}
          className="w-(--anchor-width) gap-0 p-0"
        >
          <ul
            id={listboxId}
            role="listbox"
            aria-label={t("Teammates who can view this")}
            className="max-h-60 overflow-y-auto p-1"
          >
            {candidates.length === 0 ? (
              <li className="px-2 py-2 text-sm text-muted-foreground">
                {isFetching ? t("Searching...") : t("No teammates who can view this match.")}
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
                    <span className="truncate text-xs text-muted-foreground">
                      {candidate.emailAddress}
                    </span>
                  ) : null}
                </li>
              ))
            )}
          </ul>
        </PopoverContent>
      </Popover>
    </div>
  );
}

function shareSuccessMessage(t: TranslateFn, result: ShareRecordResult): string {
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

function PersonAvatar({ user }: { user: RecordShareUser }) {
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

function SharedWithList({ record }: { record: SharedRecord }) {
  const t = useT();
  const { data: shares, isLoading, isError } = useQuery(recordShare.list(record.kind, record.id));

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
      <p className="text-sm text-destructive">{t("Couldn't load who this is shared with.")}</p>
    );
  }

  if (!shares || shares.length === 0) {
    return <p className="text-sm text-muted-foreground">{t("Not shared with anyone yet.")}</p>;
  }

  return (
    <ul className="flex max-h-64 flex-col gap-4 overflow-y-auto" aria-label={t("Shared with")}>
      {shares.map((share) => (
        <SharedWithRow key={share.id} share={share} record={record} />
      ))}
    </ul>
  );
}

function SharedWithRow({ share, record }: { share: RecordShare; record: SharedRecord }) {
  const t = useT();
  const queryClient = useQueryClient();
  // Taking a share back shows in the list itself: the row goes.
  const removal = useApiMutation({
    mutationFn: () => unshareRecord(record.kind, record.id, share.id),
    resourceName: "Share",
    scope: { id: `unshare:${share.id}` },
    onSuccess: (shares) => {
      queryClient.setQueryData(recordShare.list(record.kind, record.id).queryKey, shares);
    },
  });
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
                <span className="truncate text-sm text-muted-foreground">
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
      {share.removable && (
        <Button
          type="button"
          size="sm"
          variant="ghost"
          isLoading={removal.isPending}
          aria-label={t("Remove {0}", name)}
          onClick={() => removal.mutate()}
        >
          {t("Remove")}
        </Button>
      )}
    </li>
  );
}
