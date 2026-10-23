import { preloadable } from "@/lib/lazy-component";
import type { Extension } from "@codemirror/state";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useState } from "react";

const loadTariffVim = preloadable(() => import("./tariff-vim"));

/**
 * Vim mode's extension once it has loaded, while `enabled`; null otherwise. The module is read
 * the first time someone turns Vim keys on, never for anyone who has not. If it cannot be read,
 * the editor keeps its own keys and the person is told why Vim's are missing.
 */
export function useTariffVim(enabled: boolean): Extension | null {
  const t = useT();
  const [extension, setExtension] = useState<Extension | null>(null);

  useEffect(() => {
    if (!enabled) return;
    let live = true;
    loadTariffVim().then(
      (module) => {
        if (live) setExtension(module.tariffVim());
      },
      () => {
        if (live) toast.error(t("Vim keys could not be loaded. The editor's own keys still work."));
      },
    );
    return () => {
      live = false;
    };
  }, [enabled, t]);

  return enabled ? extension : null;
}
