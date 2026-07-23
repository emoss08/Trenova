import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { Check, Copy } from "lucide-react";
import { Button } from "./ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "./ui/tooltip";

export function FormCopyButton({ rowId }: { rowId: string }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-xxs"
            onClick={(e) => {
              e.stopPropagation();
              void copy(rowId);
            }}
          >
            {!isCopied ? <Copy className="size-2" /> : <Check className="size-2" />}
          </Button>
        }
      />
      <TooltipContent>Copy Row ID</TooltipContent>
    </Tooltip>
  );
}
