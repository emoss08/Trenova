import {
  updateFactoringRemittanceLine,
  type FactoringRemittance,
} from "@/lib/graphql/factoring-remittance";
import { invalidateFactoringQueries } from "@/lib/queries/factoring-roots";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type {
  FactoringRemittanceLineAction,
  FactoringRemittanceLineKind,
} from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { toast } from "sonner";

export type LineUpdate = {
  lineId: string;
  action: FactoringRemittanceLineAction;
  itemId?: string;
  kind?: FactoringRemittanceLineKind;
  reason?: string;
};

/** Resolves one line against the remittance's current version. */
export function useLineUpdate(remittance: FactoringRemittance, onDone?: () => void) {
  const t = useT();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (update: LineUpdate) =>
      updateFactoringRemittanceLine({
        remittanceId: remittance.id,
        version: remittance.version,
        lineId: update.lineId,
        action: update.action,
        itemId: update.itemId ?? null,
        kind: update.kind ?? null,
        reason: update.reason ?? null,
      }),
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onDone?.();
    },
    onError: (error) => {
      invalidateFactoringQueries(queryClient);
      const described = describeError(error);
      toast.error(t("The line was not changed"), {
        description: described.detail ?? described.message,
      });
    },
  });
}
