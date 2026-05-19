import { Badge } from "@/components/ui/badge";
import type { EDIMessage } from "@/types/edi";
import { DatabaseIcon } from "lucide-react";
import { formatUnix } from "../../../edi-display-utils";

export default function InspectorHeader({
  message,
  messageId,
}: {
  message?: EDIMessage;
  messageId: string;
}) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-3 border-b p-4">
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <DatabaseIcon className="size-4 text-muted-foreground" />
          <h2 className="truncate text-base font-semibold">
            Message {message?.transactionControlNumber ?? messageId}
          </h2>
          {message ? (
            <Badge variant={message.status === "Generated" ? "active" : "inactive"}>
              {message.status}
            </Badge>
          ) : null}
        </div>
        <div className="mt-1 text-sm text-muted-foreground">
          {message
            ? `${message.transactionSet} ${message.direction} generated ${formatUnix(message.generatedAt)}`
            : "Loading message details."}
        </div>
      </div>
      {message ? (
        <div className="flex flex-wrap gap-2 text-xs">
          <Badge variant="outline" className="font-mono">
            ISA {message.interchangeControlNumber}
          </Badge>
          <Badge variant="outline" className="font-mono">
            GS {message.groupControlNumber}
          </Badge>
          <Badge variant="outline" className="font-mono">
            ST {message.transactionControlNumber}
          </Badge>
        </div>
      ) : null}
    </div>
  );
}
