import {
  AiToolCall,
  AiToolCallContent,
  AiToolCallHeader,
  AiToolCallOutput,
} from "@/components/elements/ai-tool-call";
import { DateTimePicker } from "@/components/fields/date-field/datetime-picker";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { TextShimmer } from "@/components/ui/text-shimmer";
import {
  loadConversation,
  saveConversation,
  type PersistedChatMessage,
} from "@/lib/import-chat-store";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type {
  ConversationStatus,
  ImportAssistantChatMessage,
  ImportAssistantSuggestion,
  ImportAssistantToolCallRecord,
} from "@/types/document";
import { ArrowUpIcon, CheckCircle2Icon, InfoIcon, SparklesIcon } from "lucide-react";
import { m } from "motion/react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ReconciliationState } from "./types";

const TOOL_LABELS: Record<string, string> = {
  search_customers: "Searching customers",
  search_locations: "Searching locations",
  search_service_types: "Searching service types",
  search_shipment_types: "Searching shipment types",
  search_formula_templates: "Searching rating methods",
  accept_field: "Accepting field",
  accept_all_confident: "Accepting confident fields",
  set_field_value: "Setting field",
  set_required_field: "Setting required field",
};

type ToolCallState = {
  name: string;
  callId: string;
  status: "running" | "completed" | "error";
  result?: string;
};

type ChatMessage = Pick<ImportAssistantChatMessage, "id" | "role" | "text"> & {
  id: string;
  role: "user" | "assistant";
  text: string;
  toolCalls?: ToolCallState[];
  suggestions?: ImportAssistantSuggestion[];
};

type AIActivityPanelProps = {
  documentId: string;
  state: ReconciliationState;
  onAcceptField: (key: string) => void;
  onAcceptAllConfident: () => void;
  onEditField: (key: string, value: unknown) => void;
  onSetRequiredField?: (fieldKey: string, value: string) => void;
  onSetStopLocation?: (stopIndex: number, locationId: string) => void;
  onSetStopSchedule?: (stopIndex: number, windowStart: string, windowEnd?: string) => void;
  onSetShipmentField?: (field: string, value: string) => void;
  onCreateShipment?: () => void;
  onShipmentCreated?: (shipmentId: string) => void;
  lastCreateError?: string | null;
  onClearCreateError?: () => void;
  requiredFieldValues: {
    customerId: string;
    serviceTypeId: string;
    shipmentTypeId: string;
    formulaTemplateId: string;
  };
};

function getTextValue(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function toLocalToolCalls(toolCalls: ImportAssistantToolCallRecord[]): ToolCallState[] {
  return toolCalls.map((toolCall, index) => ({
    name: toolCall.name,
    callId: toolCall.callId || `${toolCall.name}-${index}`,
    status:
      toolCall.status === "error"
        ? "error"
        : toolCall.status === "running"
          ? "running"
          : "completed",
    result: toolCall.output,
  }));
}

function SuggestionButton({
  suggestion,
  onSend,
  onAction,
}: {
  suggestion: ImportAssistantSuggestion;
  onSend: (text: string) => Promise<void>;
  onAction?: (action: string) => void;
}) {
  const [isInputOpen, setIsInputOpen] = useState(false);
  const [inputVal, setInputVal] = useState("");
  const [selectedDate, setSelectedDate] = useState<Date | undefined>();
  const localInputRef = useRef<HTMLInputElement>(null);

  if (suggestion.type === "date") {
    if (isInputOpen) {
      return (
        <div className="space-y-1.5">
          <DateTimePicker
            dateTime={selectedDate}
            setDateTime={(date) => setSelectedDate(date)}
            placeholder={suggestion.placeholder || "e.g. 3/15 0800 or t+1 1400"}
            className="h-8 text-xs"
          />
          <div className="flex gap-1.5">
            <Button
              variant="outline"
              size="sm"
              className="h-7 flex-1 px-2 text-2xs"
              onClick={() => {
                if (selectedDate) {
                  const isoDate = selectedDate.toISOString();
                  void onSend(suggestion.prompt + " " + isoDate);
                  setIsInputOpen(false);
                  setSelectedDate(undefined);
                }
              }}
              disabled={!selectedDate}
            >
              {suggestion.submitLabel || "Confirm"}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 px-2 text-2xs"
              onClick={() => {
                setIsInputOpen(false);
                setSelectedDate(undefined);
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      );
    }

    return (
      <button
        type="button"
        onClick={() => setIsInputOpen(true)}
        className="rounded-md border border-dashed bg-background px-2.5 py-1.5 text-left text-2xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        {suggestion.label}
      </button>
    );
  }

  if (suggestion.type === "action" && suggestion.action) {
    return (
      <button
        type="button"
        onClick={() => onAction?.(suggestion.action!)}
        className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-1.5 text-left text-2xs font-medium text-emerald-600 transition-colors hover:bg-emerald-500/20 dark:text-emerald-400"
      >
        {suggestion.label}
      </button>
    );
  }

  if (suggestion.type === "input") {
    if (isInputOpen) {
      return (
        <div className="flex gap-1.5">
          <Input
            ref={localInputRef}
            value={inputVal}
            onChange={(e) => setInputVal(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && inputVal.trim()) {
                void onSend(suggestion.prompt + inputVal.trim());
                setIsInputOpen(false);
                setInputVal("");
              }
              if (e.key === "Escape") {
                setIsInputOpen(false);
                setInputVal("");
              }
            }}
            placeholder={suggestion.placeholder || "Type here..."}
            className="h-7 flex-1 text-2xs"
            autoFocus
          />
          <Button
            variant="outline"
            size="sm"
            className="h-7 shrink-0 px-2 text-2xs"
            onClick={() => {
              if (inputVal.trim()) {
                void onSend(suggestion.prompt + inputVal.trim());
                setIsInputOpen(false);
                setInputVal("");
              }
            }}
            disabled={!inputVal.trim()}
          >
            {suggestion.submitLabel || "Confirm"}
          </Button>
        </div>
      );
    }

    return (
      <button
        type="button"
        onClick={() => {
          setIsInputOpen(true);
          requestAnimationFrame(() => localInputRef.current?.focus());
        }}
        className="rounded-md border border-dashed bg-background px-2.5 py-1.5 text-left text-2xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        {suggestion.label}
      </button>
    );
  }

  return (
    <button
      type="button"
      onClick={() => void onSend(suggestion.prompt)}
      className="rounded-md border bg-background px-2.5 py-1.5 text-left text-2xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
    >
      {suggestion.label}
    </button>
  );
}

function ToolResultSummary({ result, name }: { result: string; name: string }) {
  try {
    const data = JSON.parse(result);

    if (name === "search_customers" && data.customers) {
      const customers = data.customers as Array<{ id: string; name: string }>;
      if (customers.length === 0)
        return <span className="text-muted-foreground">No customers found</span>;
      return (
        <div className="space-y-1">
          {customers.map((c) => (
            <div key={c.id} className="text-2xs">
              {c.name}
            </div>
          ))}
          <div className="text-2xs text-muted-foreground">{data.total} total</div>
        </div>
      );
    }

    if (name === "search_locations" && (data.locations || data.availableLocations)) {
      const exactLocations = data.locations as
        | Array<{
            id: string;
            name: string;
            code: string;
            addressLine1?: string;
            city?: string;
            postalCode?: string;
          }>
        | undefined;
      const fallbackLocations = data.availableLocations as
        | Array<{
            id: string;
            name: string;
            code: string;
            addressLine1?: string;
            city?: string;
            postalCode?: string;
          }>
        | undefined;
      const locations = (exactLocations?.length ? exactLocations : fallbackLocations) ?? [];
      if (locations.length === 0)
        return <span className="text-muted-foreground">No locations found</span>;
      const label = data.noExactMatch ? "Available locations:" : "";
      return (
        <div className="space-y-1">
          {label && <div className="text-2xs text-muted-foreground">{label}</div>}
          {locations.map((l) => (
            <div key={l.id} className="text-2xs">
              {l.code} — {l.name}
              {l.city && <span className="text-muted-foreground"> ({l.city})</span>}
            </div>
          ))}
        </div>
      );
    }

    if (name === "search_service_types" && data.serviceTypes) {
      const types = data.serviceTypes as Array<{ id: string; name: string; code: string }>;
      if (types.length === 0)
        return <span className="text-muted-foreground">No service types found</span>;
      return (
        <div className="space-y-1">
          {types.map((t) => (
            <div key={t.id} className="text-2xs">
              {t.code} — {t.name}
            </div>
          ))}
        </div>
      );
    }

    if (name === "search_shipment_types" && data.shipmentTypes) {
      const types = data.shipmentTypes as Array<{ id: string; name: string }>;
      if (types.length === 0)
        return <span className="text-muted-foreground">No shipment types found</span>;
      return (
        <div className="space-y-1">
          {types.map((t) => (
            <div key={t.id} className="text-2xs">
              {t.name}
            </div>
          ))}
        </div>
      );
    }

    if (name === "search_formula_templates" && data.formulaTemplates) {
      const templates = data.formulaTemplates as Array<{ id: string; name: string }>;
      if (templates.length === 0)
        return <span className="text-muted-foreground">No rating methods found</span>;
      return (
        <div className="space-y-1">
          {templates.map((t) => (
            <div key={t.id} className="text-2xs">
              {t.name}
            </div>
          ))}
        </div>
      );
    }

    if (data.accepted)
      return <span className="text-2xs text-emerald-500">Accepted: {data.accepted}</span>;
    if (data.set)
      return (
        <span className="text-2xs">
          Set {data.set} = {data.value}
        </span>
      );
    if (data.set_required) {
      const label = data.entity_id
        ? `Set to ${data.label || data.entity_id}`
        : `Set ${data.set_required}`;
      return <span className="text-2xs text-emerald-500">{label}</span>;
    }

    return <span className="text-2xs text-muted-foreground">{result.slice(0, 100)}</span>;
  } catch {
    return <span className="text-2xs text-muted-foreground">{result.slice(0, 100)}</span>;
  }
}

export default function AIActivityPanel({
  documentId,
  state,
  onAcceptField,
  onAcceptAllConfident,
  onEditField,
  onSetRequiredField,
  onSetStopLocation,
  onSetStopSchedule,
  onSetShipmentField,
  onCreateShipment,
  onShipmentCreated,
  lastCreateError,
  onClearCreateError,
  requiredFieldValues,
}: AIActivityPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversationId, setConversationId] = useState<string | undefined>();
  const [conversationStatus, setConversationStatus] = useState<ConversationStatus>("Active");
  const [statusReason, setStatusReason] = useState("");
  const [isStreaming, setIsStreaming] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const hasSentInitial = useRef(false);
  const [hasHydrated, setHasHydrated] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const streamingMsgId = useRef<string | null>(null);
  const lastContentMsgId = useRef<string | null>(null);

  const isConversationClosed =
    conversationStatus === "Completed" || conversationStatus === "Superseded";

  const filledRequired = [
    requiredFieldValues.customerId,
    requiredFieldValues.serviceTypeId,
    requiredFieldValues.shipmentTypeId,
    requiredFieldValues.formulaTemplateId,
  ].filter(Boolean).length;
  const isReady = filledRequired === 4;

  const scrollToBottom = useCallback(() => {
    requestAnimationFrame(() => {
      scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: "smooth" });
    });
  }, []);

  const reconciliationStateForAPI = useMemo(() => {
    const fields: Record<string, unknown> = {};
    for (const [key, field] of Object.entries(state.fields)) {
      fields[key] = {
        label: field.label,
        value: field.value,
        confidence: field.confidence,
        status: field.status,
      };
    }
    return fields;
  }, [state.fields]);

  const processActions = useCallback(
    (
      actions: Array<{
        type: string;
        fieldKey: string;
        value: string;
        metadata?: Record<string, unknown>;
      }>,
    ) => {
      for (const action of actions) {
        switch (action.type) {
          case "accept_field":
            onAcceptField(action.fieldKey);
            break;
          case "accept_all_confident":
            onAcceptAllConfident();
            break;
          case "set_field":
            onEditField(action.fieldKey, action.value);
            break;
          case "set_required_field":
            onSetRequiredField?.(action.fieldKey, action.value);
            break;
          case "set_stop_location":
            onSetStopLocation?.(Number.parseInt(action.fieldKey, 10), action.value);
            break;
          case "set_stop_schedule": {
            const windowEnd = (action.metadata?.window_end as string) || undefined;
            onSetStopSchedule?.(Number.parseInt(action.fieldKey, 10), action.value, windowEnd);
            break;
          }
          case "set_shipment_field":
            onSetShipmentField?.(action.fieldKey, action.value);
            break;
          case "shipment_created":
            onShipmentCreated?.(action.value);
            break;
          case "create_shipment":
            onCreateShipment?.();
            break;
        }
      }
    },
    [
      onAcceptField,
      onAcceptAllConfident,
      onEditField,
      onSetRequiredField,
      onSetStopLocation,
      onSetStopSchedule,
      onSetShipmentField,
      onCreateShipment,
      onShipmentCreated,
    ],
  );

  const sendMessage = useCallback(
    async (text: string) => {
      if (!text.trim() || isStreaming || isConversationClosed) return;

      const userMsg: ChatMessage = { id: `user-${Date.now()}`, role: "user", text: text.trim() };
      const assistantId = `assist-${Date.now()}`;
      streamingMsgId.current = assistantId;

      setMessages((prev) => [
        ...prev,
        userMsg,
        { id: assistantId, role: "assistant", text: "", toolCalls: [] },
      ]);
      setIsStreaming(true);
      setInputValue("");
      scrollToBottom();

      const stopsForAPI = state.stops.map((stop, i) => ({
        index: i,
        role: stop.role,
        sequence: stop.sequence,
        name: stop.name.value,
        addressLine1: stop.addressLine1.value,
        city: stop.city.value,
        state: stop.state.value,
        postalCode: stop.postalCode.value,
        date: stop.date.value,
        timeWindow: stop.timeWindow.value,
        locationId: stop.locationId || "",
        hasLocation: !!stop.locationId,
        hasValidDate: getTextValue(stop.date.value).includes("-"),
        confidence: stop.confidence,
      }));

      // Add a summary so the AI knows the overall stop status at a glance
      const stopsSummary = {
        totalStops: stopsForAPI.length,
        stopsWithLocation: stopsForAPI.filter((s) => s.hasLocation).length,
        stopsWithDate: stopsForAPI.filter((s) => s.hasValidDate).length,
        stopsNeedingAttention: stopsForAPI.filter((s) => !s.hasLocation || !s.hasValidDate).length,
      };

      const shipmentData: Record<string, unknown> = {};
      for (const [key, field] of Object.entries(state.fields)) {
        if (field.value != null && field.value !== "") {
          shipmentData[key] = field.value;
        }
      }

      await apiService.documentService.chatWithImportAssistantStream(
        documentId,
        {
          message: text.trim(),
          conversationId,
          reconciliationState: reconciliationStateForAPI,
          requiredFields: requiredFieldValues,
          stops: stopsForAPI,
          shipmentData: { ...shipmentData, _stopsSummary: stopsSummary },
        },
        {
          onTextDelta: (delta) => {
            lastContentMsgId.current = streamingMsgId.current;
            setMessages((prev) =>
              prev.map((m) =>
                m.id === streamingMsgId.current ? { ...m, text: m.text + delta } : m,
              ),
            );
            scrollToBottom();
          },
          onNewMessage: () => {
            // Only create a new bubble if the current one already has text.
            // If it only has tool calls (no text), keep accumulating so
            // tool calls and their resulting text stay in the same bubble.
            setMessages((prev) => {
              const current = prev.find((m) => m.id === streamingMsgId.current);
              if (current && current.text.length === 0) {
                return prev;
              }
              const newId = `assist-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`;
              streamingMsgId.current = newId;
              lastContentMsgId.current = newId;
              return [...prev, { id: newId, role: "assistant", text: "", toolCalls: [] }];
            });
            scrollToBottom();
          },
          onToolCallStart: (name, callId) => {
            lastContentMsgId.current = streamingMsgId.current;
            setMessages((prev) =>
              prev.map((m) =>
                m.id === streamingMsgId.current
                  ? {
                      ...m,
                      toolCalls: [
                        ...(m.toolCalls ?? []),
                        { name, callId, status: "running" as const },
                      ],
                    }
                  : m,
              ),
            );
            scrollToBottom();
          },
          onToolCallDone: (_name, callId, status, result, actions) => {
            setMessages((prev) =>
              prev.map((m) => {
                const tcIdx = m.toolCalls?.findIndex((tc) => tc.callId === callId);
                if (tcIdx == null || tcIdx < 0) return m;
                const updated = [...(m.toolCalls ?? [])];
                updated[tcIdx] = {
                  ...updated[tcIdx],
                  status: status === "error" ? ("error" as const) : ("completed" as const),
                  result,
                };
                return { ...m, toolCalls: updated };
              }),
            );
            if (actions) processActions(actions);
            scrollToBottom();
          },
          onSuggestions: (newSuggestions) => {
            const targetId = lastContentMsgId.current ?? streamingMsgId.current;
            setMessages((prev) =>
              prev.map((message) =>
                message.id === targetId ? { ...message, suggestions: newSuggestions } : message,
              ),
            );
          },
          onDone: (newConversationId, actions) => {
            setConversationId(newConversationId);
            if (actions) processActions(actions);
            setIsStreaming(false);
            streamingMsgId.current = null;
            lastContentMsgId.current = null;
            scrollToBottom();
          },
          onError: (message) => {
            setMessages((prev) =>
              prev.map((m) =>
                m.id === streamingMsgId.current
                  ? { ...m, text: message || "Something went wrong." }
                  : m,
              ),
            );
            setIsStreaming(false);
            streamingMsgId.current = null;
            lastContentMsgId.current = null;
          },
        },
      );
    },
    [
      documentId,
      conversationId,
      reconciliationStateForAPI,
      requiredFieldValues,
      isStreaming,
      isConversationClosed,
      processActions,
      scrollToBottom,
      state.fields,
      state.stops,
    ],
  );

  // When shipment creation fails, parse errors and send readable summary to AI
  useEffect(() => {
    if (lastCreateError && !isStreaming) {
      onClearCreateError?.();

      // Try to parse validation errors into human-readable format
      let readableError = lastCreateError;
      try {
        const errors = JSON.parse(lastCreateError) as Array<{ path: string[]; message: string }>;
        if (Array.isArray(errors)) {
          const summary = errors
            .map((e) => {
              const path = e.path?.join(".") ?? "unknown";
              return `- ${path}: ${e.message}`;
            })
            .join("\n");
          readableError = `Validation failed:\n${summary}`;
        }
      } catch {
        // Not JSON, use as-is
      }

      void sendMessage(
        `Shipment creation failed. Here are the issues:\n${readableError}\n\nPlease help me fix these one at a time.`,
      );
    }
  }, [lastCreateError, isStreaming, onClearCreateError, sendMessage]);

  // Load persisted conversation on mount
  useEffect(() => {
    if (!documentId) return;

    setMessages([]);
    setConversationId(undefined);
    setConversationStatus("Active");
    setStatusReason("");
    setInputValue("");
    setIsStreaming(false);
    hasSentInitial.current = false;
    setHasHydrated(false);
    streamingMsgId.current = null;
    lastContentMsgId.current = null;

    const hydrateFromIndexedDB = () => {
      void loadConversation(documentId).then((saved) => {
        if (saved && saved.messages.length > 0) {
          hasSentInitial.current = true;
          setMessages(saved.messages as ChatMessage[]);
          if (saved.conversationId) setConversationId(saved.conversationId);
        }
        setHasHydrated(true);
      });
    };

    void apiService.documentService
      .getImportAssistantHistory(documentId)
      .then((history) => {
        if (history.status) setConversationStatus(history.status);
        if (history.statusReason) setStatusReason(history.statusReason);

        if (history.messages.length > 0) {
          hasSentInitial.current = true;
          setMessages(
            history.messages.map((message) => ({
              id: message.id,
              role: message.role,
              text: message.text,
              toolCalls: toLocalToolCalls(message.toolCalls),
              suggestions: message.suggestions,
            })),
          );
          if (history.conversationId) {
            setConversationId(history.conversationId);
          }
          setHasHydrated(true);
          return;
        }

        hydrateFromIndexedDB();
      })
      .catch(() => {
        hydrateFromIndexedDB();
      });
  }, [documentId]);

  // Persist conversation after each completed message exchange
  useEffect(() => {
    if (!documentId || !hasHydrated || isStreaming || messages.length === 0) return;

    const persistable: PersistedChatMessage[] = messages
      .filter((m) => m.text.length > 0)
      .map((m) => ({
        id: m.id,
        role: m.role,
        text: m.text,
        toolCalls: m.toolCalls,
        suggestions: m.suggestions,
      }));

    void saveConversation(documentId, conversationId, persistable);
  }, [documentId, conversationId, hasHydrated, messages, isStreaming]);

  // Auto-send initial message once, including extracted context
  useEffect(() => {
    if (
      !documentId ||
      !hasHydrated ||
      hasSentInitial.current ||
      isStreaming ||
      isConversationClosed ||
      messages.length > 0
    ) {
      return;
    }

    hasSentInitial.current = true;
    const shipperName = getTextValue(state.fields.shipper?.value);
    const initialMsg = shipperName
      ? `Help me complete this shipment. The extracted shipper name is "${shipperName}".`
      : "Help me complete this shipment.";
    void sendMessage(initialMsg);
  }, [
    documentId,
    hasHydrated,
    isStreaming,
    isConversationClosed,
    messages.length,
    sendMessage,
    state.fields.shipper?.value,
  ]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        void sendMessage(inputValue);
      }
    },
    [inputValue, sendMessage],
  );

  const adjustHeight = useCallback(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 100)}px`;
  }, []);

  useEffect(() => {
    adjustHeight();
  }, [inputValue, adjustHeight]);

  // Find last assistant message index
  let lastAssistantIdx = -1;
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === "assistant") {
      lastAssistantIdx = i;
      break;
    }
  }

  return (
    <div className="flex h-full flex-col border-l">
      {/* Header */}
      <div className="shrink-0 border-b px-3 py-2.5">
        <div className="flex items-center gap-2">
          <SparklesIcon className="size-3.5 text-muted-foreground" />
          <span className="text-xs font-medium">AI Assistant</span>
        </div>
        <div className="mt-2">
          <div className="mb-1 flex items-center justify-between text-2xs text-muted-foreground">
            <span>{isReady ? "Ready to create" : `${4 - filledRequired} fields remaining`}</span>
          </div>
          <div className="h-0.5 overflow-hidden rounded-full bg-muted">
            <m.div
              className={cn("h-full rounded-full", isReady ? "bg-emerald-500" : "bg-foreground/40")}
              animate={{ width: `${(filledRequired / 4) * 100}%` }}
              transition={{ duration: 0.4 }}
            />
          </div>
        </div>
      </div>

      {/* Messages */}
      <div ref={scrollRef} className="flex-1 overflow-y-auto">
        <div className="flex flex-col gap-2 p-2.5">
          {isConversationClosed && (
            <div className="flex items-center gap-2 rounded-md bg-muted px-2.5 py-1.5">
              <InfoIcon className="size-3 text-muted-foreground" />
              <span className="text-2xs text-muted-foreground">
                {conversationStatus === "Completed" && statusReason === "shipment_created"
                  ? "This import has been completed."
                  : conversationStatus === "Superseded"
                    ? "This conversation was superseded by a re-extraction."
                    : "This conversation is no longer active."}
              </span>
            </div>
          )}

          {isReady && !isConversationClosed && messages.length > 0 && (
            <m.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="flex items-center gap-2 rounded-md bg-emerald-500/10 px-2.5 py-1.5"
            >
              <CheckCircle2Icon className="size-3 text-emerald-500" />
              <span className="text-2xs text-emerald-600 dark:text-emerald-400">
                Ready to create shipment
              </span>
            </m.div>
          )}

          {messages.map((msg, idx) => (
            <m.div
              key={msg.id}
              initial={{ opacity: 0, y: 4 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.15 }}
            >
              {msg.role === "user" ? (
                <div className="flex justify-end">
                  <div className="max-w-[85%] rounded-lg bg-foreground/[0.06] px-2.5 py-1 text-xs text-muted-foreground">
                    {msg.text}
                  </div>
                </div>
              ) : (
                <div>
                  {/* Tool calls */}
                  {msg.toolCalls && msg.toolCalls.length > 0 && (
                    <div className="mb-1.5 space-y-1">
                      {msg.toolCalls.map((tc) => (
                        <AiToolCall
                          key={tc.callId}
                          name={TOOL_LABELS[tc.name] ?? tc.name}
                          state={tc.status}
                          className="text-2xs"
                        >
                          <AiToolCallHeader className="px-2 py-1 text-2xs [&_span.font-mono]:text-2xs [&_svg]:size-3 [&>div:first-child]:size-4 [&>div:first-child]:rounded [&>span]:py-0 [&>span]:text-2xs" />
                          {tc.result && tc.status === "completed" && (
                            <AiToolCallContent className="[&>div]:space-y-1 [&>div]:p-1.5">
                              <AiToolCallOutput className="[&>div]:p-1.5 [&>div]:text-2xs [&>span]:text-2xs">
                                <ToolResultSummary result={tc.result} name={tc.name} />
                              </AiToolCallOutput>
                            </AiToolCallContent>
                          )}
                        </AiToolCall>
                      ))}
                    </div>
                  )}

                  {/* Message text — streams in real-time */}
                  <div className="text-[13px] leading-relaxed text-foreground">
                    {msg.text}
                    {isStreaming && msg.id === streamingMsgId.current && msg.text.length > 0 && (
                      <span className="ml-0.5 inline-block h-[13px] w-[1.5px] animate-pulse bg-foreground/40 align-text-bottom" />
                    )}
                  </div>

                  {/* Thinking shimmer when streaming with no text yet */}
                  {isStreaming &&
                    msg.id === streamingMsgId.current &&
                    msg.text.length === 0 &&
                    msg.toolCalls?.length === 0 && (
                      <TextShimmer as="span" className="text-[13px]" duration={2}>
                        Thinking
                      </TextShimmer>
                    )}

                  {/* Suggestions — only on latest assistant message, after streaming completes */}
                  {idx === lastAssistantIdx &&
                    !isStreaming &&
                    (msg.suggestions?.length ?? 0) > 0 && (
                      <m.div
                        initial={{ opacity: 0, y: 4 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: 0.15 }}
                        className="mt-1.5 flex flex-col gap-1"
                      >
                        {msg.suggestions?.map((s) => (
                          <SuggestionButton
                            key={s.label}
                            suggestion={s}
                            onSend={sendMessage}
                            onAction={(action) => {
                              if (action === "create_shipment") onCreateShipment?.();
                              if (action === "review_details") {
                                // Scroll the reconciliation panel to top
                                document
                                  .querySelector('[data-slot="scroll-area-viewport"]')
                                  ?.scrollTo({ top: 0, behavior: "smooth" });
                              }
                            }}
                          />
                        ))}
                      </m.div>
                    )}
                </div>
              )}
            </m.div>
          ))}
        </div>
      </div>

      {/* Input */}
      <div className="shrink-0 border-t p-2">
        <div className="flex items-end gap-1.5 rounded-lg border bg-background px-3 py-1.5 focus-within:ring-1 focus-within:ring-ring">
          <textarea
            ref={inputRef}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={isConversationClosed ? "Conversation closed" : "Ask the assistant..."}
            disabled={isStreaming || isConversationClosed}
            rows={1}
            className="flex-1 resize-none bg-transparent text-xs leading-relaxed outline-none placeholder:text-muted-foreground/40 disabled:opacity-50"
            style={{ minHeight: 22, maxHeight: 100 }}
          />
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={() => void sendMessage(inputValue)}
            disabled={isStreaming || isConversationClosed || !inputValue.trim()}
            className="mb-px shrink-0"
          >
            <ArrowUpIcon className="size-3.5" />
          </Button>
        </div>
      </div>
    </div>
  );
}
