import { Button } from "@trenova/shared/components/ui/button";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { CopyIcon } from "lucide-react";
import type { InspectorContext } from "../inspector-context";
import { controlNumberText } from "../inspector-utils";
import InspectorGrid from "./inspector-grid";

export { controlNumberText };

export default function ControlNumbersTab({ context }: { context: InspectorContext }) {
  const { copy } = useCopyToClipboard();

  return (
    <div>
      <div className="mb-3 flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          onClick={() => void copy(context.controlCopyText, { withToast: true })}
        >
          <CopyIcon className="size-4" />
          Copy
        </Button>
      </div>
      <InspectorGrid rows={context.controlRows} />
    </div>
  );
}
