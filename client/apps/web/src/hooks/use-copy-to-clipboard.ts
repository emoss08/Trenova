// * CREDIT https://github.com/openstatusHQ/data-table-filters/blob/main/src/hooks/use-copy-to-clipboard.ts
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useState } from "react";
import { toast } from "sonner";

export function useCopyToClipboard() {
  const t = useT();

  const [text, setText] = useState<string | null>(null);

  const copy = useCallback(
    async (
      text: string,
      { timeout, withToast }: { timeout?: number; withToast?: boolean } = {
        timeout: 3000,
        withToast: false,
      },
    ) => {
      if (!navigator?.clipboard) {
        console.warn("Clipboard not supported");
        return false;
      }

      try {
        await navigator.clipboard.writeText(text);
        setText(text);

        if (timeout) {
          setTimeout(() => {
            setText(null);
          }, timeout);
        }

        if (withToast) {
          toast.success(t("Copied to clipboard"));
        }

        return true;
      } catch (error) {
        console.warn("Copy failed", error);
        setText(null);
        return false;
      }
    },
    [t],
  );

  return { text, copy, isCopied: text !== null };
}
