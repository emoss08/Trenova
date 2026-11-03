import { translate } from "@trenova/shared/i18n/runtime";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { AccessorialChargeAutocompleteField } from "@/components/autocomplete-fields";
import { DocumentSignatureDialog } from "@/components/documents/document-signature-dialog";
import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { documentNeedsSignature } from "@/lib/document-signature";
import type { ExtractedField, Extraction } from "@/lib/document-reading/extraction-payload";
import { billingQueue } from "@/lib/queries/billing-queue";
import { documentReading } from "@/lib/queries/document-reading";
import { apiService } from "@/services/api";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Form } from "@trenova/shared/components/ui/form";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import type { Document } from "@trenova/shared/types/document";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState, type ReactNode } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { shipmentDocumentsQuery } from "../billing-queue-queries";
import { useBulkRun } from "./use-bulk-run";
import { useInvalidateBillingQueue } from "./use-billing-queue-invalidate";

export type ReadingState = "loading" | "failed" | "unread" | "reading" | "read";

/** Words in a read field's key or label that say which fact it is. */
const FACT_HINTS: Record<keyof FactsForm, readonly string[]> = {
  signedBy: ["signed", "signature", "consignee", "receivedby"],
  pieces: ["piece", "pallet", "count", "quantity"],
  osd: ["osd", "overshort", "damage", "exception"],
  osdNote: ["note", "remark"],
  seal: ["seal"],
  arrivedAt: ["arriv", "timein", "intime", "checkin"],
  departedAt: ["depart", "timeout", "outtime", "checkout"],
  amount: ["amount", "total", "paid"],
  reference: ["receipt", "ticket", "reference", "invoicenumber"],
  chargeId: [],
};

function folded(text: string): string {
  return text.toLowerCase().replace(/[^a-z0-9]/gu, "");
}

/** The read field each fact was taken from, so focusing the fact finds it on the page. */
function linkFacts(fields: readonly ExtractedField[]) {
  const links = new Map<keyof FactsForm, ExtractedField>();
  const used = new Set<string>();
  for (const fact of Object.keys(FACT_HINTS) as (keyof FactsForm)[]) {
    const hit = fields.find(
      (field) =>
        !used.has(field.key) &&
        FACT_HINTS[fact].some((hint) => folded(`${field.key} ${field.label}`).includes(hint)),
    );
    if (hit) {
      links.set(fact, hit);
      used.add(hit.key);
    }
  }
  return { links, rest: fields.filter((field) => !used.has(field.key) && !field.stop) };
}

function factsSummary(facts: Document["deliveryFacts"], t: TranslateFn): string {
  if (!facts) return "";
  const parts = [
    facts.signedBy ? t("signed by {0}", facts.signedBy) : t("no signature"),
    facts.pieces,
    facts.seal ? t("seal {0}", facts.seal) : "",
    facts.osd && facts.osd !== "None" ? t("{0} noted", facts.osd.toLowerCase()) : "",
    facts.amount ? t("total {0}", facts.amount) : "",
  ];
  return parts.filter(Boolean).join(", ");
}

function novaSummary(fields: readonly ExtractedField[], t: TranslateFn): string {
  return fields
    .slice(0, 5)
    .map((field) => t("{0} {1}", field.label.toLowerCase(), field.value))
    .join(", ");
}

const factsFormSchema = z
  .object({
    signedBy: z.string().trim().max(200),
    pieces: z.string().trim().max(50),
    osd: z.enum(["None", "Over", "Short", "Damaged"]),
    osdNote: z.string().trim().max(500),
    seal: z.string().trim().max(100),
    arrivedAt: z.number().nullable(),
    departedAt: z.number().nullable(),
    amount: z
      .string()
      .trim()
      .regex(/^(\d{1,9}(\.\d{1,2})?)?$/u, {
        error: () => translate("Enter an amount like 185.00"),
      }),
    reference: z.string().trim().max(100),
    chargeId: z.string().nullable(),
  })
  .refine((v) => v.departedAt == null || v.arrivedAt == null || v.departedAt >= v.arrivedAt, {
    path: ["departedAt"],
    error: () => translate("Departure can't be before arrival"),
  })
  .refine((v) => !v.chargeId || v.amount !== "", {
    path: ["amount"],
    error: () => translate("A receipt that bills a charge needs its amount"),
  });
type FactsForm = z.infer<typeof factsFormSchema>;

function formValuesOf(document: Document): FactsForm {
  const facts = document.deliveryFacts;
  return {
    signedBy: facts?.signedBy ?? "",
    pieces: facts?.pieces ?? "",
    osd: facts?.osd ?? "None",
    osdNote: facts?.osdNote ?? "",
    seal: facts?.seal ?? "",
    arrivedAt: facts?.arrivedAt ?? null,
    departedAt: facts?.departedAt ?? null,
    amount: facts?.amount ?? "",
    reference: facts?.reference ?? "",
    chargeId: facts?.chargeId ?? null,
  };
}

/**
 * One document beside its page, as the review's right side: what Nova read from
 * it, the facts billing checks (who signed, pieces, over, short or damaged, seal,
 * in and out times, and for a receipt its amount and the charge it bills as) with
 * a Nova mark on every value read from the page, what to do when the page holds the
 * item up, whether it goes out with the invoice, and the file's own actions.
 */
export function BillingQueueDocumentDetails({
  document,
  item,
  reading,
  readingState,
  placing = false,
  selected,
  hot,
  onPoint,
  onSelectField,
  packet,
  onReadStarted,
  onNext,
  onDeleted,
}: {
  document: Document;
  item: BillingQueueItem | undefined;
  reading?: Extraction;
  readingState: ReadingState;
  /** Nova's facts are in, and the conversion that places them on the page is still running. */
  placing?: boolean;
  selected: string | null;
  hot: string | null;
  onPoint: (key: string | null) => void;
  onSelectField: (key: string) => void;
  packet?: {
    included: boolean;
    editable: boolean;
    saving: boolean;
    onToggle: (include: boolean) => void;
  };
  /** Told once a read was asked for, so the review watches for it to finish. */
  onReadStarted: () => void;
  onNext?: () => void;
  onDeleted: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const invalidate = useInvalidateBillingQueue();
  const bulk = useBulkRun();
  const { allowed: canUpdate } = usePermission(Resource.Document, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.Document, Operation.Delete);
  const [signing, setSigning] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);
  const facts = document.deliveryFacts;
  const readFields = new Set(facts?.readFields ?? []);
  const fields = reading?.fields ?? [];
  const { links, rest } = linkFacts(fields);
  const stop = fields.find((field) => field.stop)?.stop ?? null;

  const form = useForm<FactsForm>({
    resolver: zodResolver(factsFormSchema),
    defaultValues: formValuesOf(document),
  });
  const osd = useWatch({ control: form.control, name: "osd" });

  const refreshDocuments = () => {
    if (item?.shipmentId) {
      void queryClient.invalidateQueries({
        queryKey: shipmentDocumentsQuery(item.shipmentId).queryKey,
      });
    }
  };

  const save = useApiMutation({
    mutationFn: (values: FactsForm) =>
      apiService.documentService.setDeliveryFacts(
        document.id,
        {
          signedBy: values.signedBy,
          pieces: values.pieces,
          osd: values.osd === "None" ? null : values.osd,
          osdNote: values.osdNote,
          seal: values.seal,
          arrivedAt: values.arrivedAt,
          departedAt: values.departedAt,
          amount: values.amount || null,
          reference: values.reference,
          chargeId: values.chargeId,
        },
        document.version,
      ),
    resourceName: "Document",
    form,
    onSuccess: (next) => {
      form.reset(formValuesOf(next));
      refreshDocuments();
      invalidate();
    },
  });

  const resolve = useApiMutation({
    mutationFn: ({ issueId, optionKey }: { issueId: string; optionKey: string }) =>
      apiService.billingQueueService.resolveIssue(item?.id ?? "", issueId, optionKey),
    resourceName: "BillingQueueItem",
    onSuccess: (next) => {
      if (item && next.review) {
        queryClient.setQueryData(billingQueue.get(item.id).queryKey, next);
      }
      invalidate();
    },
  });

  const readNow = useApiMutation({
    mutationFn: () => apiService.documentService.reextract(document.id),
    resourceName: "Document",
    onSuccess: () => {
      onReadStarted();
      refreshDocuments();
      void queryClient.invalidateQueries({ queryKey: documentReading.get(document.id).queryKey });
    },
  });

  const remove = useApiMutation({
    mutationFn: () => apiService.documentService.delete(document.id),
    resourceName: "Document",
    onSuccess: () => {
      refreshDocuments();
      invalidate();
      onDeleted();
    },
  });

  const replace = useApiMutation({
    mutationFn: (file: File) =>
      apiService.documentService.upload({
        file,
        resourceId: document.resourceId,
        resourceType: document.resourceType,
        lineageId: document.lineageId,
        documentTypeId: document.documentTypeId ?? undefined,
      }),
    resourceName: "Document",
    onSuccess: () => {
      refreshDocuments();
      invalidate();
    },
  });

  const unsigned = documentNeedsSignature(document) && document.signatureStatus !== "Signed";
  const unbilled = (item?.review?.issues ?? []).find(
    (issue) =>
      issue.code === "UnbilledAccessorial" && !issue.resolvedAt && issue.subjectKey === document.id,
  );
  const addOption = unbilled?.options.find((option) => option.effect.kind === "add");
  const skipOption = unbilled?.options.find((option) => option.effect.kind === "accept");
  const customer = item?.billToCustomer?.name ?? item?.shipment?.customer?.name;
  const state = unsigned
    ? { label: t("Unsigned"), tone: "warning" as const }
    : unbilled
      ? { label: t("Not billed"), tone: "warning" as const }
      : documentNeedsSignature(document)
        ? { label: t("Signed"), tone: "success" as const }
        : { label: t("Received"), tone: "success" as const };
  const osdOptions = [
    { value: "None", label: t("None noted") },
    { value: "Over", label: t("Over") },
    { value: "Short", label: t("Short") },
    { value: "Damaged", label: t("Damaged") },
  ];
  const isRead = (fact: keyof FactsForm) => readFields.has(fact) || links.has(fact);
  const focusFact = (fact: keyof FactsForm) => {
    const link = links.get(fact);
    if (link) onSelectField(link.key);
  };
  const cell = (fact: keyof FactsForm, label: string, control: ReactNode) => (
    <FactCell
      key={fact}
      label={label}
      read={isRead(fact)}
      lit={Boolean(
        links.get(fact) && (selected === links.get(fact)?.key || hot === links.get(fact)?.key),
      )}
      onFocus={() => focusFact(fact)}
      onPoint={(on) => onPoint(on ? (links.get(fact)?.key ?? null) : null)}
    >
      {control}
    </FactCell>
  );

  return (
    <section
      aria-label={t("Document details")}
      className="bg-card flex h-full min-w-0 flex-col text-sm"
    >
      <div className="border-border flex flex-col gap-1 border-b px-4.5 py-4">
        <div className="flex items-center gap-2">
          <h2 className="text-foreground text-xl font-semibold">
            {document.documentType?.name ?? document.originalName}
          </h2>
          <span
            className={cn(
              "rounded-xs px-1.5 py-0.5 text-xs font-medium",
              state.tone === "warning"
                ? "bg-warning-subtle text-warning-foreground"
                : "bg-success-subtle text-success-foreground",
            )}
          >
            {state.label}
          </span>
        </div>
        <span className="text-foreground-muted">
          {t(
            "{0} · {1} · {2, plural, one {# page} other {# pages}}",
            document.originalName,
            formatUnixInUserTimezone(document.createdAt, {
              month: "short",
              day: "numeric",
              hour: "numeric",
              minute: "2-digit",
            }),
            Math.max(1, reading?.pageCount ?? 1),
          )}
        </span>
      </div>

      <ScrollArea maskVariant="card" className="min-h-0 flex-1">
        <div className="flex flex-col gap-4 px-4.5 py-4">
          <NovaReading
            state={readingState}
            placing={placing}
            kind={reading?.kindLabel ?? ""}
            summary={fields.length > 0 ? novaSummary(fields, t) : factsSummary(facts, t)}
            reading={readNow.isPending}
            onRead={() => readNow.mutate()}
          />

          <Form
            onSubmit={(event) => void form.handleSubmit((values) => save.mutate(values))(event)}
            className="flex flex-col gap-3"
          >
            <div className="grid grid-cols-2 gap-x-3 gap-y-2.5">
              <FactCell label={t("Type")} read={false}>
                <ReadOnly value={document.documentType?.name ?? t("Not classified")} />
              </FactCell>
              <FactCell label={t("Stop")} read={Boolean(stop)}>
                <ReadOnly
                  value={
                    stop
                      ? t("Stop {0} · {1} {2}", stop.sequence, stop.place || "—", stop.role)
                      : t("Not read from the page")
                  }
                />
              </FactCell>
              {cell(
                "signedBy",
                t("Signed by"),
                <InputField
                  control={form.control}
                  name="signedBy"
                  placeholder={t("Nobody signed")}
                  disabled={!canUpdate}
                />,
              )}
              {cell(
                "pieces",
                t("Pieces"),
                <InputField
                  control={form.control}
                  name="pieces"
                  placeholder={t("22 pallets")}
                  disabled={!canUpdate}
                />,
              )}
              {cell(
                "osd",
                t("OS&D"),
                <SelectField
                  control={form.control}
                  name="osd"
                  options={osdOptions}
                  isReadOnly={!canUpdate}
                />,
              )}
              {cell(
                "seal",
                t("Seal"),
                <InputField
                  control={form.control}
                  name="seal"
                  placeholder={t("4471902")}
                  disabled={!canUpdate}
                />,
              )}
              {cell(
                "arrivedAt",
                t("In"),
                <AutoCompleteDateTimeField
                  control={form.control}
                  name="arrivedAt"
                  label=""
                  placeholder={t("Arrival time")}
                  disabled={!canUpdate}
                />,
              )}
              {cell(
                "departedAt",
                t("Out"),
                <AutoCompleteDateTimeField
                  control={form.control}
                  name="departedAt"
                  label=""
                  placeholder={t("Departure time")}
                  disabled={!canUpdate}
                />,
              )}
              {osd !== "None" || facts?.osdNote
                ? cell(
                    "osdNote",
                    t("Exception note"),
                    <InputField
                      control={form.control}
                      name="osdNote"
                      placeholder={t("2 cartons crushed")}
                      disabled={!canUpdate}
                    />,
                  )
                : null}
              {cell(
                "reference",
                t("Receipt number"),
                <InputField
                  control={form.control}
                  name="reference"
                  placeholder={t("LMP-5521")}
                  disabled={!canUpdate}
                />,
              )}
              {cell(
                "amount",
                t("Receipt amount"),
                <InputField
                  control={form.control}
                  name="amount"
                  placeholder={t("185.00")}
                  inputMode="decimal"
                  disabled={!canUpdate}
                />,
              )}
              {cell(
                "chargeId",
                t("Bills as"),
                <AccessorialChargeAutocompleteField
                  control={form.control}
                  name="chargeId"
                  placeholder={t("Choose an accessorial")}
                  clearable
                  disabled={!canUpdate}
                />,
              )}
              {rest.map((field) => (
                <FactCell
                  key={field.key}
                  label={field.label}
                  read
                  lit={selected === field.key || hot === field.key}
                  onFocus={() => onSelectField(field.key)}
                  onPoint={(on) => onPoint(on ? field.key : null)}
                >
                  <ReadOnly value={field.value} doubtful={field.needsLook} />
                </FactCell>
              ))}
            </div>
            {canUpdate && form.formState.isDirty ? (
              <div className="flex justify-end gap-2">
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  onClick={() => form.reset(formValuesOf(document))}
                >
                  {t("Discard")}
                </Button>
                <Button
                  type="submit"
                  size="sm"
                  isLoading={save.isPending}
                  loadingText={t("Saving...")}
                >
                  {t("Save changes")}
                </Button>
              </div>
            ) : null}
          </Form>

          {unsigned && item ? (
            <div className="border-warning-border bg-warning-subtle flex flex-col gap-2 rounded-md border p-3">
              <p className="text-warning-foreground leading-4.5">
                <span className="font-semibold">{t("The consignee didn't sign.")}</span>{" "}
                {customer
                  ? t("{0} requires a signed POD, so this item can't be approved yet.", customer)
                  : t("A signed POD is required, so this item can't be approved yet.")}
              </p>
              <div className="flex flex-wrap gap-1.5">
                <Button
                  size="sm"
                  isLoading={bulk.isStarting}
                  onClick={() => void bulk.start("RequestPaperwork", [item.id])}
                >
                  {t("Ask the driver for a signed copy")}
                </Button>
                {canUpdate ? (
                  <Button size="sm" variant="outline" onClick={() => setSigning(true)}>
                    {t("Record a signature")}
                  </Button>
                ) : null}
              </div>
            </div>
          ) : null}

          {unbilled ? (
            <div className="border-border flex flex-col gap-2 rounded-md border p-3">
              <p className="leading-4.5">
                <span className="font-semibold">{unbilled.summary}</span>
                {unbilled.reasoning ? (
                  <span className="text-foreground-muted"> {unbilled.reasoning}</span>
                ) : null}
              </p>
              <div className="flex gap-1.5">
                {addOption ? (
                  <Button
                    size="sm"
                    disabled={resolve.isPending}
                    onClick={() =>
                      resolve.mutate({ issueId: unbilled.id, optionKey: addOption.key })
                    }
                  >
                    {addOption.label ||
                      (addOption.amount != null
                        ? t("Bill it · {0}", formatCurrency(Number(addOption.amount)))
                        : t("Bill it"))}
                  </Button>
                ) : null}
                {skipOption ? (
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={resolve.isPending}
                    onClick={() =>
                      resolve.mutate({ issueId: unbilled.id, optionKey: skipOption.key })
                    }
                  >
                    {t("Not billable")}
                  </Button>
                ) : null}
              </div>
            </div>
          ) : null}

          {packet ? (
            <div className="border-border flex flex-col gap-2 border-t pt-3.5">
              <label className="flex items-center gap-2">
                {packet.saving ? (
                  <span
                    role="status"
                    aria-label={t("Saving")}
                    className="grid size-4 shrink-0 place-items-center"
                  >
                    <Spinner className="size-3.5" />
                  </span>
                ) : (
                  <Checkbox
                    checked={packet.included}
                    disabled={!packet.editable}
                    onCheckedChange={(checked) => packet.onToggle(checked === true)}
                  />
                )}
                {t("Send with the invoice packet")}
              </label>
            </div>
          ) : null}
        </div>
      </ScrollArea>

      <footer className="border-border flex flex-wrap items-center gap-1.5 border-t px-4.5 py-2.5">
        {canDelete ? (
          confirmDelete ? (
            <>
              <Button
                size="sm"
                variant="destructive"
                isLoading={remove.isPending}
                onClick={() => remove.mutate()}
              >
                {t("Delete this file")}
              </Button>
              <Button size="sm" variant="ghost" onClick={() => setConfirmDelete(false)}>
                {t("Keep it")}
              </Button>
            </>
          ) : (
            <Button
              size="sm"
              variant="ghost"
              className="text-destructive"
              onClick={() => setConfirmDelete(true)}
            >
              {t("Delete")}
            </Button>
          )
        ) : null}
        {canUpdate ? (
          <>
            <Button
              size="sm"
              variant="outline"
              isLoading={replace.isPending}
              onClick={() => fileInput.current?.click()}
            >
              {t("Replace file")}
            </Button>
            <input
              ref={fileInput}
              type="file"
              className="hidden"
              aria-hidden
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) replace.mutate(file);
                event.target.value = "";
              }}
            />
          </>
        ) : null}
        <span className="flex-1" />
        {onNext ? (
          <Button size="sm" variant="outline" onClick={onNext}>
            {t("Next document")}
          </Button>
        ) : null}
      </footer>

      <DocumentSignatureDialog
        document={signing ? document : null}
        open={signing}
        onOpenChange={setSigning}
      />
    </section>
  );
}

/** What Nova made of the page, or how to have it read. */
function NovaReading({
  state,
  placing,
  kind,
  summary,
  reading,
  onRead,
}: {
  state: ReadingState;
  placing: boolean;
  kind: string;
  summary: string;
  reading: boolean;
  onRead: () => void;
}) {
  const t = useT();

  if (state === "loading") {
    return <Skeleton className="h-14 w-full" />;
  }
  if (state === "failed") {
    return (
      <Alert size="sm" variant="warning">
        <AlertDescription>
          {t("Nova's reading of this document couldn't be loaded.")}
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="border-border bg-muted flex gap-2.5 rounded-md border px-3 py-2.5">
      <NovaBadge
        size="sm"
        className="mt-0.5"
        spin={state === "reading" || reading ? "always" : "none"}
      />
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        {state === "read" && summary ? (
          <>
            <p className="leading-4.5">
              <span className="font-semibold">{t("Nova read this page")}</span>
              <span> · {summary}</span>
            </p>
            <span className="text-foreground-muted text-xs">
              {placing
                ? t("Filled the fields below · placing them on the page…")
                : t("Filled the fields below · check anything that looks wrong")}
            </span>
          </>
        ) : state === "read" ? (
          <>
            <p className="leading-4.5">
              <span className="font-semibold">{t("Nova read this page")}</span>
              <span>
                {" · "}
                {kind && kind !== "Other" && kind !== "Document"
                  ? t(
                      "it reads as a {0}, but nothing on it fills these fields.",
                      kind.toLowerCase(),
                    )
                  : t("nothing on it reads as freight paperwork, so there's nothing to fill in.")}
              </span>
            </p>
            <div>
              <Button size="xs" variant="ghost" isLoading={reading} onClick={onRead}>
                {t("Read it again")}
              </Button>
            </div>
          </>
        ) : state === "reading" ? (
          <p className="leading-4.5">{t("Nova is reading this page…")}</p>
        ) : (
          <>
            <p className="leading-4.5">{t("Nova hasn't read this page yet.")}</p>
            <div>
              <Button size="xs" variant="outline" isLoading={reading} onClick={onRead}>
                {t("Read with Nova")}
              </Button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function FactCell({
  label,
  read,
  lit = false,
  onFocus,
  onPoint,
  children,
}: {
  label: string;
  read: boolean;
  lit?: boolean;
  onFocus?: () => void;
  onPoint?: (on: boolean) => void;
  children: ReactNode;
}) {
  const t = useT();

  return (
    // oxlint-disable-next-line jsx-a11y/no-static-element-interactions -- pointing at a fact lights where it was read; the control inside is the keyboard's way in
    <div
      className={cn(
        "flex min-w-0 flex-col gap-1 rounded-sm",
        lit && "ring-brand/40 ring-2 ring-offset-2",
      )}
      onFocusCapture={onFocus}
      onMouseEnter={() => onPoint?.(true)}
      onMouseLeave={() => onPoint?.(false)}
    >
      <span className="text-foreground-muted flex items-center gap-1.25 text-xs font-medium">
        {label}
        {read ? <NovaBadge size="xs" className="size-2.25" label={t("Read by Nova")} /> : null}
      </span>
      {children}
    </div>
  );
}

function ReadOnly({ value, doubtful = false }: { value: string; doubtful?: boolean }) {
  return (
    <span
      className={cn(
        "ui-field bg-muted flex h-7 items-center truncate px-2",
        doubtful ? "text-warning-foreground" : "text-foreground",
      )}
      title={value}
    >
      {value}
    </span>
  );
}
