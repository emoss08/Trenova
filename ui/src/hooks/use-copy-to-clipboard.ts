/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

// * CREDIT https://github.com/openstatusHQ/data-table-filters/blob/main/src/hooks/use-copy-to-clipboard.ts
import { useCallback, useState } from "react";
import { toast } from "sonner";

export function useCopyToClipboard() {
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
          toast.success("Copied to clipboard");
        }

        return true;
      } catch (error) {
        console.warn("Copy failed", error);
        setText(null);
        return false;
      }
    },
    [],
  );

  return { text, copy, isCopied: text !== null };
}
