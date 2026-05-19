import { Button } from "@/components/ui/button";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { downloadJsonFile } from "@/lib/utils";
import type { EDIMessage } from "@/types/edi";
import { json } from "@codemirror/lang-json";
import { EditorView } from "@codemirror/view";
import CodeMirror from "@uiw/react-codemirror";
import { CopyIcon, DownloadIcon } from "lucide-react";
import { useMemo } from "react";
import { buildMessageJsonFilename } from "../../utils/edi-message-utils";
import type { useEditorTheme } from "../../components/designer-shared";

export default function PayloadTab({
  message,
  editorTheme,
}: {
  message: EDIMessage;
  editorTheme: ReturnType<typeof useEditorTheme>;
}) {
  const { copy } = useCopyToClipboard();
  const payloadJson = useMemo(
    () => JSON.stringify(message.payloadSnapshot ?? {}, null, 2),
    [message.payloadSnapshot],
  );

  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
      <div className="mb-2 flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          onClick={() => void copy(payloadJson, { withToast: true })}
        >
          <CopyIcon className="size-4" />
          Copy
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={() =>
            downloadJsonFile(buildMessageJsonFilename(message), message.payloadSnapshot)
          }
        >
          <DownloadIcon className="size-4" />
          Download
        </Button>
      </div>
      <CodeMirror
        value={payloadJson}
        editable={false}
        basicSetup={{ lineNumbers: true, foldGutter: true }}
        extensions={[json(), EditorView.lineWrapping]}
        theme={editorTheme}
        className="min-h-0 overflow-auto rounded-md border text-xs"
      />
    </div>
  );
}
