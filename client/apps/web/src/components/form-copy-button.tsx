import { useT } from "@trenova/shared/i18n/use-t";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { CheckIcon, Copy01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";

export function FormCopyButton({ rowId }: { rowId: string }) {
  const t = useT();

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
            {!isCopied ? <Copy01Icon className="size-2" /> : <CheckIcon className="size-2" />}
          </Button>
        }
      />
      <TooltipContent>{t("Copy row ID")}</TooltipContent>
    </Tooltip>
  );
}
