import { PageAssistant } from "@/components/assistant/page-assistant";
import type { PageBinding, PageRequest } from "@/components/assistant/message-thread";
import { apiService } from "@/services/api";
import type { PageDraftEdit } from "@/types/page-draft";
import { useQuery } from "@tanstack/react-query";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronRightIcon, HistoryIcon } from "lucide-react";
import { m } from "motion/react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { earlierConversation, type EarlierConversation } from "./earlier-conversation";
import {
  applyImportDraftEdit,
  createFailureSummary,
  importAssistantConversationKey,
  importDraftFromState,
  type ImportDraftHandlers,
  type ImportRequiredValues,
} from "./import-draft";
import type { ReconciliationState } from "./types";

const REQUIRED_COUNT = 4;

type AIActivityPanelProps = {
  documentId: string;
  state: ReconciliationState;
  onAcceptField: (key: string) => void;
  onAcceptAllConfident: () => void;
  onEditField: (key: string, value: unknown) => void;
  onSetRequiredField?: (fieldKey: string, value: string) => void;
  onSetStopLocation?: (stopIndex: number, locationId: string) => void;
  onSetStopSchedule?: (stopIndex: number, windowStart: string, windowEnd?: string) => void;
  lastCreateError?: string | null;
  onClearCreateError?: () => void;
  requiredFieldValues: ImportRequiredValues;
};

function ignore() {}

/**
 * The import assistant beside the document: the shared conversation about
 * this document, with the reconciliation on screen riding on every question.
 * What it changes — a field accepted, a stop matched, a customer chosen —
 * lands on this page as if the person had clicked it; nothing is saved until
 * they create the shipment. A new location is a proposal they approve.
 */
export default function AIActivityPanel({
  documentId,
  state,
  onAcceptField,
  onAcceptAllConfident,
  onEditField,
  onSetRequiredField,
  onSetStopLocation,
  onSetStopSchedule,
  lastCreateError,
  onClearCreateError,
  requiredFieldValues,
}: AIActivityPanelProps) {
  const t = useT();

  const filledRequired = [
    requiredFieldValues.customerId,
    requiredFieldValues.serviceTypeId,
    requiredFieldValues.shipmentTypeId,
    requiredFieldValues.formulaTemplateId,
  ].filter((value) => value.trim() !== "").length;
  const isReady = filledRequired === REQUIRED_COUNT;

  const handlers = useMemo<ImportDraftHandlers>(
    () => ({
      acceptField: onAcceptField,
      acceptAllConfident: onAcceptAllConfident,
      editField: onEditField,
      setRequiredField: onSetRequiredField ?? ignore,
      setStopLocation: onSetStopLocation ?? ignore,
      setStopSchedule: onSetStopSchedule ?? ignore,
    }),
    [
      onAcceptAllConfident,
      onAcceptField,
      onEditField,
      onSetRequiredField,
      onSetStopLocation,
      onSetStopSchedule,
    ],
  );

  const stopCount = state.stops.length;
  const onDraftEdit = useCallback(
    (edit: PageDraftEdit) => {
      if (!applyImportDraftEdit(edit, handlers, stopCount)) {
        toast.warning(t("The assistant made a change that no longer fits this page"), {
          description: t("Nothing was changed. Ask it again if you still want the change."),
        });
      }
    },
    [handlers, stopCount, t],
  );

  const readDraft = useCallback(
    () => importDraftFromState(state, requiredFieldValues),
    [requiredFieldValues, state],
  );

  const page = useMemo<PageBinding>(
    () => ({ surface: "shipment_import", readDraft, onDraftEdit }),
    [onDraftEdit, readDraft],
  );

  const open = useCallback(
    () => apiService.documentService.openImportAssistantThread(documentId),
    [documentId],
  );

  // A failed create is handed to the assistant as the person's own question,
  // once, so it can walk them through what the shipment was refused over.
  const pageRequest = useMemo<PageRequest | null>(
    () =>
      lastCreateError
        ? {
            key: lastCreateError,
            text: t(
              "Creating the shipment failed:\n{0}\n\nHelp me fix these one at a time.",
              createFailureSummary(lastCreateError),
            ),
          }
        : null,
    [lastCreateError, t],
  );
  const onPageRequestSent = useCallback(() => onClearCreateError?.(), [onClearCreateError]);

  const shipper = state.fields.shipper?.value;
  const openingQuestion =
    typeof shipper === "string" && shipper.trim() !== ""
      ? t('Help me complete this shipment. The document names the shipper "{0}".', shipper.trim())
      : t("Help me complete this shipment.");

  const historyQuery = useQuery({
    queryKey: ["import-assistant-history", documentId],
    queryFn: () => apiService.documentService.getImportAssistantHistory(documentId),
    staleTime: Number.POSITIVE_INFINITY,
    retry: false,
  });
  const earlier = earlierConversation(historyQuery.data);

  return (
    <PageAssistant
      className="h-full border-l"
      conversationKey={importAssistantConversationKey(documentId)}
      open={open}
      page={page}
      openingQuestion={openingQuestion}
      pageRequest={pageRequest}
      onPageRequestSent={onPageRequestSent}
      header={
        <>
          <div className="flex flex-col gap-1">
            <span className="text-2xs text-muted-foreground">
              {isReady
                ? t("Ready to create")
                : t("{0} fields remaining", REQUIRED_COUNT - filledRequired)}
            </span>
            <div className="bg-muted h-0.5 overflow-hidden rounded-full">
              <m.div
                className={cn("h-full rounded-full", isReady ? "bg-success" : "bg-foreground/40")}
                animate={{ width: `${(filledRequired / REQUIRED_COUNT) * 100}%` }}
                transition={{ duration: 0.4 }}
              />
            </div>
          </div>
          {earlier && <EarlierConversationNotice conversation={earlier} />}
        </>
      }
    />
  );
}

/**
 * What was said about this document before the assistant moved onto the
 * shared conversation, kept readable for a release. It cannot be continued.
 */
function EarlierConversationNotice({ conversation }: { conversation: EarlierConversation }) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger className="text-2xs text-muted-foreground hover:text-foreground ui-focus-ring flex items-center gap-1 rounded-sm">
        <ChevronRightIcon className={cn("size-3 transition-transform", open && "rotate-90")} />
        <HistoryIcon className="size-3" />
        {conversation.reason === "superseded"
          ? t("Earlier conversation, ended by a re-extraction")
          : t("Earlier conversation, finished")}
      </CollapsibleTrigger>
      <CollapsibleContent>
        <ol className="bg-muted/40 mt-1.5 flex max-h-56 flex-col gap-1.5 overflow-y-auto rounded-md border p-2">
          {conversation.messages.map((message) => (
            <li
              key={message.id}
              className={cn(
                "text-xs leading-relaxed whitespace-pre-wrap",
                message.role === "user" ? "text-muted-foreground" : "text-foreground",
              )}
            >
              <span className="font-medium">
                {message.role === "user" ? t("You") : t("Assistant")}
              </span>
              {": "}
              {message.text}
            </li>
          ))}
        </ol>
        <p className="text-2xs text-muted-foreground mt-1">
          {t("This conversation is read-only. Ask the assistant below to continue.")}
        </p>
      </CollapsibleContent>
    </Collapsible>
  );
}
