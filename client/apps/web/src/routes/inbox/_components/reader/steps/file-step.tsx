import { ShipmentAutocomplete } from "@/components/autocomplete-fields";
import type { InboundAttachment, InboundMessageDetail, TriageAction } from "@/lib/graphql/inbox";
import type { SelectOption } from "@/lib/graphql/select-options";
import { describeApiError } from "@/lib/api-error-message";
import { documentTypeName, documentTypeNamed } from "@/lib/triage/document-types";
import { useTriageStore } from "@/stores/triage-store";
import { CheckIcon, EyeIcon, LayersThree01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { useTriageActions } from "../../use-triage-actions";
import { useTriageDocumentTypes } from "../../use-triage-data";
import { readTypeId } from "../file-chips";
import { useStepSubmit } from "../reader-panel";
import { splitRanges } from "../split-pages";
import { StepBody, StepError, StepField, StepFooter } from "./step-parts";

/** The value a page's type select uses for "Don't file". */
const SKIP = "__skip";
const NO_TYPES: readonly SelectOption[] = [];

function TypeSelect({
  label,
  value,
  documentTypes,
  allowSkip,
  onChange,
  className,
}: {
  label: string;
  value: string;
  documentTypes: readonly SelectOption[];
  allowSkip: boolean;
  onChange: (value: string) => void;
  className?: string;
}) {
  const t = useT();
  const items = [
    ...documentTypes.map((type) => ({ value: type.id, label: documentTypeName(type) })),
    ...(allowSkip ? [{ value: SKIP, label: t("Don't file") }] : []),
  ];

  return (
    <Select value={value} items={items} onValueChange={(next) => next !== null && onChange(next)}>
      <SelectTrigger aria-label={label} className={cn("h-7 w-full text-sm", className)}>
        <SelectValue placeholder={t("Choose a type")} />
      </SelectTrigger>
      <SelectContent>
        {items.map((item) => (
          <SelectItem key={item.value} value={item.value}>
            {item.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

function FiledResult({
  message,
  attachment,
}: {
  message: InboundMessageDetail;
  attachment: InboundAttachment;
}) {
  const t = useT();
  const actions = useTriageActions();

  return (
    <StepBody className="gap-1.5">
      {attachment.splitDocuments.length > 0 && (
        <ul className="flex flex-col gap-0.5 text-sm">
          {attachment.splitDocuments.map((document) => (
            <li key={document.documentId} className="flex items-center gap-2">
              <CheckIcon aria-hidden className="text-success size-3" strokeWidth={2.4} />
              <span className="min-w-0 flex-1 truncate">{document.documentType}</span>
              <span className="text-foreground-subtle font-mono text-xs tabular-nums">
                {document.fromPage === document.toPage
                  ? t("Page {0}", document.fromPage)
                  : t("Pages {0}–{1}", document.fromPage, document.toPage)}
              </span>
            </li>
          ))}
        </ul>
      )}
      <button
        type="button"
        className="text-foreground-subtle hover:text-foreground ui-focus-ring w-fit rounded-sm text-xs"
        onClick={() => actions.unfile(message.id, attachment.id)}
      >
        {t("Undo")}
      </button>
    </StepBody>
  );
}

/**
 * Filing a file from the panel: as one document of a type on a shipment, or
 * split into its pages, each with its own type (or left out). The type and
 * the shipment start from what Nova read; File sends it.
 */
export function FileStep({
  message,
  attachment,
  action,
  initialSplit,
}: {
  message: InboundMessageDetail;
  attachment: InboundAttachment;
  action: TriageAction;
  initialSplit: boolean;
}) {
  const t = useT();
  const actions = useTriageActions();
  const documentTypes = useTriageDocumentTypes().data?.results ?? NO_TYPES;
  const pages = attachment.pages;
  const canSplit = pages.length > 1;
  const [split, setSplit] = useState(initialSplit && canSplit);
  const [typeId, setTypeId] = useState<string | null>(null);
  const [pageTypes, setPageTypes] = useState<Record<number, string>>({});
  const [shipmentId, setShipmentId] = useState(
    action.shipment?.id ?? message.matchedShipment?.id ?? "",
  );
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const wholeType = typeId ?? readTypeId(attachment, documentTypes) ?? documentTypes[0]?.id ?? "";
  const chosenPageTypes = useMemo(
    () =>
      pages.map((page) => {
        const chosen = pageTypes[page.number];
        if (chosen !== undefined) {
          return chosen === SKIP ? null : chosen;
        }
        return documentTypeNamed(documentTypes, page.suggestedType)?.id ?? null;
      }),
    [pages, pageTypes, documentTypes],
  );
  const ranges = useMemo(
    () =>
      splitRanges(
        pages.map((page) => page.number),
        chosenPageTypes,
      ),
    [pages, chosenPageTypes],
  );

  const ready =
    !pending &&
    attachment.documentId !== null &&
    shipmentId !== "" &&
    (split ? ranges.length > 0 : wholeType !== "");

  const submit = () => {
    if (!ready) {
      return;
    }
    setPending(true);
    setError(null);
    actions
      .file(message.id, {
        attachmentId: attachment.id,
        shipmentId,
        ...(split ? { pages: ranges } : { documentTypeId: wholeType }),
      })
      .catch((failure: unknown) => setError(describeApiError(failure)))
      .finally(() => setPending(false));
  };
  useStepSubmit(action.id, ready ? submit : null);

  if (action.done) {
    return <FiledResult message={message} attachment={attachment} />;
  }

  const shipmentField = (
    <StepField label={t("On")}>
      <ShipmentAutocomplete
        label={t("Shipment")}
        value={shipmentId}
        onValueChange={setShipmentId}
        placeholder={t("Shipment")}
        triggerClassName="h-7 text-sm"
      />
    </StepField>
  );

  return (
    <StepBody>
      {canSplit && (
        <button
          type="button"
          aria-pressed={split}
          onClick={() => setSplit((value) => !value)}
          className={cn(
            "text-foreground-muted hover:text-foreground hover:border-foreground ui-focus-ring flex h-7 items-center gap-1.75 rounded-md border px-2.25 text-sm",
            split ? "border-border-strong text-foreground" : "border-border-strong border-dashed",
          )}
        >
          <LayersThree01Icon aria-hidden className="size-3" />
          {split
            ? t("File as one document")
            : t(
                "{0, plural, one {Split into # document} other {Split into # documents}}",
                pages.length,
              )}
          <span className="text-foreground-subtle ml-auto font-mono text-2xs tabular-nums">
            {t("{0, plural, one {# page} other {# pages}}", pages.length)}
          </span>
        </button>
      )}
      {split ? (
        <>
          <ol className="flex flex-col gap-1">
            {pages.map((page, index) => (
              <li
                key={page.number}
                className="animate-rise grid grid-cols-[28px_minmax(0,1fr)_minmax(0,9rem)] items-center gap-2"
              >
                <span
                  aria-hidden
                  className="bg-sunken border-border text-foreground-subtle flex h-8.5 items-end justify-center rounded-xs border pb-0.75 font-mono text-3xs"
                >
                  {page.number}
                </span>
                <span className="truncate text-sm">{page.label}</span>
                <TypeSelect
                  label={t("Page {0} as", page.number)}
                  value={chosenPageTypes[index] ?? SKIP}
                  documentTypes={documentTypes}
                  allowSkip
                  onChange={(value) =>
                    setPageTypes((current) => ({ ...current, [page.number]: value }))
                  }
                />
              </li>
            ))}
          </ol>
          {shipmentField}
        </>
      ) : (
        <div className="flex flex-col gap-1.5">
          <StepField label={t("As")}>
            <TypeSelect
              label={t("File as")}
              value={wholeType}
              documentTypes={documentTypes}
              allowSkip={false}
              onChange={setTypeId}
            />
          </StepField>
          {shipmentField}
        </div>
      )}
      {error !== null && <StepError>{error}</StepError>}
      <StepFooter>
        <Button
          size="sm"
          variant="ghost"
          onClick={() =>
            useTriageStore
              .getState()
              .openOverlay({ kind: "doc", messageId: message.id, attachmentId: attachment.id })
          }
        >
          <EyeIcon aria-hidden className="size-3" />
          {t("Preview")}
        </Button>
        <span className="flex-1" />
        <Button
          size="sm"
          disabled={!ready}
          isLoading={pending}
          loadingText={t("Filing…")}
          onClick={submit}
        >
          {split
            ? t("{0, plural, one {File # document} other {File # documents}}", ranges.length)
            : t("File")}
        </Button>
      </StepFooter>
    </StepBody>
  );
}
