import type { EDIX12EnvelopeSettings } from "@/types/edi";
import { InputBlock } from "../components/designer-shared";

export function EnvelopeEditor({
  envelope,
  onChange,
}: {
  envelope: EDIX12EnvelopeSettings;
  onChange: (envelope: EDIX12EnvelopeSettings) => void;
}) {
  const update = (key: keyof EDIX12EnvelopeSettings, value: string) => {
    onChange({ ...envelope, [key]: value });
  };
  return (
    <div className="space-y-2 rounded-md border bg-muted/30 p-2">
      <div className="text-xs font-medium">X12 Envelope</div>
      <div className="grid grid-cols-2 gap-2">
        <InputBlock
          label="ISA Sender"
          value={envelope.interchangeSenderId}
          onChange={(value) => update("interchangeSenderId", value)}
        />
        <InputBlock
          label="ISA Receiver"
          value={envelope.interchangeReceiverId}
          onChange={(value) => update("interchangeReceiverId", value)}
        />
        <InputBlock
          label="GS Sender"
          value={envelope.applicationSenderCode}
          onChange={(value) => update("applicationSenderCode", value)}
        />
        <InputBlock
          label="GS Receiver"
          value={envelope.applicationReceiverCode}
          onChange={(value) => update("applicationReceiverCode", value)}
        />
      </div>
    </div>
  );
}

export default EnvelopeEditor;
