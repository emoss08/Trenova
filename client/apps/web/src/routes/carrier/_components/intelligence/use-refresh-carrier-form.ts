import { useT } from "@trenova/shared/i18n/use-t";
import { apiService } from "@/services/api";
import type { Carrier } from "@trenova/shared/types/carrier";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { toast } from "sonner";

export function useRefreshCarrierForm(carrierId: string) {
  const t = useT();
  const queryClient = useQueryClient();
  const { control, reset } = useFormContext<Carrier>();
  const { isDirty } = useFormState({ control });

  return useCallback(async () => {
    void queryClient.invalidateQueries({ queryKey: ["carrier-list"] });

    if (isDirty) {
      toast.warning(t("The carrier record changed"), {
        description: t(
          "Intelligence updated this carrier while you have unsaved edits. Save or discard them, then reopen the carrier to see the new values.",
        ),
      });
      return;
    }

    try {
      const carrier = await apiService.carrierService.getById(carrierId);
      reset(carrier);
    } catch {
      toast.warning(t("The carrier record changed"), {
        description: t("Reopen the carrier to see the values intelligence applied."),
      });
    }
  }, [carrierId, isDirty, queryClient, reset, t]);
}
