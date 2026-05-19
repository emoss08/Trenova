import { Button } from "@/components/ui/button";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import type { EDIMessage } from "@/types/edi";
import { CopyIcon } from "lucide-react";
import InspectorGrid from "./inspector-grid";

export default function ControlNumbersTab({ message }: { message: EDIMessage }) {
  const { copy } = useCopyToClipboard();
  const text = controlNumberText(message);

  return (
    <div>
      <div className="mb-3 flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          onClick={() => void copy(text, { withToast: true })}
        >
          <CopyIcon className="size-4" />
          Copy
        </Button>
      </div>
      <InspectorGrid
        rows={[
          ["Interchange Control Number", message.interchangeControlNumber],
          ["Group Control Number", message.groupControlNumber],
          ["Transaction Control Number", message.transactionControlNumber],
          ["Segment Count", String(message.segmentCount)],
        ]}
      />
    </div>
  );
}

export function controlNumberText(message: EDIMessage) {
  return [
    `ISA: ${message.interchangeControlNumber}`,
    `GS: ${message.groupControlNumber}`,
    `ST: ${message.transactionControlNumber}`,
  ].join("\n");
}
