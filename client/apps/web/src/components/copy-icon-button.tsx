import { useT } from "@trenova/shared/i18n/use-t";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { CheckIcon, CopyIcon } from "lucide-react";

export function CopyIconButton({
  value,
  label,
  size = "icon-xs",
}: {
  value: string;
  label: string;
  size?: "icon-xs" | "icon-xxs";
}) {
  const t = useT();

  const { copy, isCopied } = useCopyToClipboard();

  return (
    <Tooltip>
      <TooltipTrigger
        render={<Button size={size} variant="ghost" onClick={() => void copy(value)} />}
      >
        {isCopied ? (
          <CheckIcon className="size-3.5 text-emerald-500" />
        ) : (
          <CopyIcon className="size-3.5" />
        )}
      </TooltipTrigger>
      <TooltipContent>{isCopied ? t("Copied") : label}</TooltipContent>
    </Tooltip>
  );
}
