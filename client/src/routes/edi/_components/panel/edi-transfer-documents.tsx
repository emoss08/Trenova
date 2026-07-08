import { JsonViewer, type JsonValue } from "@/components/elements/json-viewer";
import {
  EDIMessageAckStatusBadge,
  EDIMessageDeliveryStatusBadge,
} from "@/components/status-badge";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { formatToUserTimezone } from "@/lib/date";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { EDIMessage, EDITransfer } from "@/types/edi";
import { useQuery } from "@tanstack/react-query";
import { CopyIcon, FileJsonIcon, FileTextIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { EDIEmptyState } from "./edi-panel-primitives";
import { EDIRawX12Viewer } from "./edi-raw-x12-viewer";

const PAYLOAD_VIEW = "payload";

export function TransferDocuments({ transfer }: { transfer: EDITransfer }) {
  const { copy } = useCopyToClipboard();
  const { data, isLoading } = useQuery({
    ...queries.edi.transferMessages(transfer.id),
    enabled: !!transfer.id,
  });

  const messages = useMemo(() => data?.results ?? [], [data]);
  const [active, setActive] = useState<string>(PAYLOAD_VIEW);

  const activeMessage = messages.find((message) => message.id === active);
  const payloadJson = useMemo(
    () => JSON.stringify(transfer.tenderPayload, null, 2),
    [transfer.tenderPayload],
  );

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-1.5">
        <DocumentPill
          active={active === PAYLOAD_VIEW}
          icon={<FileJsonIcon className="size-3.5" />}
          label="Tender Payload"
          onClick={() => setActive(PAYLOAD_VIEW)}
        />
        {messages.map((message) => (
          <DocumentPill
            key={message.id}
            active={active === message.id}
            icon={<FileTextIcon className="size-3.5" />}
            label={`${message.transactionSet} · ${message.direction}`}
            onClick={() => setActive(message.id)}
          />
        ))}
        {isLoading && messages.length === 0 && (
          <span className="text-xs text-muted-foreground">Loading documents…</span>
        )}
      </div>

      {active === PAYLOAD_VIEW ? (
        <div className="rounded-lg border">
          <div className="flex items-center justify-between border-b px-3 py-2">
            <div className="text-xs font-medium text-muted-foreground">
              Normalized load tender payload
            </div>
            <Button
              type="button"
              size="xs"
              variant="outline"
              onClick={() => void copy(payloadJson, { withToast: true })}
            >
              <CopyIcon className="size-3.5" />
              Copy JSON
            </Button>
          </div>
          <div className="max-h-[28rem] overflow-auto p-3">
            <JsonViewer
              data={transfer.tenderPayload as unknown as JsonValue}
              collapsed={2}
              copyPath={false}
            />
          </div>
        </div>
      ) : activeMessage ? (
        <MessageDocument message={activeMessage} />
      ) : (
        <EDIEmptyState message="Select a document to view its contents." />
      )}
    </div>
  );
}

function MessageDocument({ message }: { message: EDIMessage }) {
  const hasPayload = !!message.payloadSnapshot;
  return (
    <div className="flex flex-col gap-3">
      <div className="grid grid-cols-2 gap-x-4 gap-y-2 rounded-lg border bg-muted/20 px-3 py-2.5 sm:grid-cols-3">
        <MetaField label="Transaction Set" value={message.transactionSet} />
        <MetaField label="Direction" value={message.direction} />
        <MetaField label="X12 Version" value={message.x12Version || "—"} />
        <MetaField
          label="ISA Control"
          value={message.interchangeControlNumber || "—"}
          mono
        />
        <MetaField label="Segments" value={String(message.segmentCount)} />
        <MetaField label="Generated" value={formatToUserTimezone(message.generatedAt)} />
      </div>
      {(message.deliveryStatus || message.ackStatus) && (
        <div className="flex flex-wrap items-center gap-2">
          {message.deliveryStatus && (
            <div className="flex items-center gap-1.5">
              <span className="text-xs text-muted-foreground">Delivery</span>
              <EDIMessageDeliveryStatusBadge status={message.deliveryStatus} />
            </div>
          )}
          {message.ackStatus && (
            <div className="flex items-center gap-1.5">
              <span className="text-xs text-muted-foreground">Ack</span>
              <EDIMessageAckStatusBadge status={message.ackStatus} />
            </div>
          )}
        </div>
      )}
      {message.rawX12 ? (
        <EDIRawX12Viewer content={message.rawX12} />
      ) : message.rawPurgedAt ? (
        <EDIEmptyState
          message={`Raw X12 was purged on ${formatToUserTimezone(message.rawPurgedAt)}.`}
        />
      ) : (
        <EDIEmptyState message="No raw X12 payload is available for this document." />
      )}
      {hasPayload && (
        <details className="rounded-lg border">
          <summary className="flex cursor-pointer list-none items-center justify-between px-3 py-2 text-xs font-medium text-muted-foreground">
            Payload snapshot
            <Badge variant="outline">JSON</Badge>
          </summary>
          <div className="max-h-96 overflow-auto border-t p-3">
            <JsonViewer
              data={message.payloadSnapshot as unknown as JsonValue}
              collapsed={2}
              copyPath={false}
            />
          </div>
        </details>
      )}
    </div>
  );
}

function MetaField({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="min-w-0">
      <div className="text-[10px] tracking-wide text-muted-foreground uppercase">{label}</div>
      <div className={cn("mt-0.5 truncate text-sm", mono && "font-mono text-xs")}>{value}</div>
    </div>
  );
}

function DocumentPill({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium transition-colors",
        active
          ? "border-primary/40 bg-primary/10 text-foreground"
          : "border-border text-muted-foreground hover:bg-accent hover:text-foreground",
      )}
    >
      {icon}
      {label}
    </button>
  );
}
