import { AccessorialChargePanel } from "@/components/accessorial-charge/accessorial-charge-panel";
import { usePermission } from "@/hooks/use-permission";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { missingChargeCode } from "@/lib/tariff-charge-code";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { TariffDiagnostic } from "@trenova/shared/types/tariff";
import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

/**
 * On a charge code the catalog doesn't have, for someone who may create accessorial charges:
 * opens the catalog's create panel with the code and the charge's label filled in, and checks
 * the source again once the charge is saved.
 */
export function AddChargeCode({
  diagnostic,
  source,
}: {
  diagnostic: TariffDiagnostic;
  source: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed } = usePermission(Resource.AccessorialCharge, Operation.Create);
  const [open, setOpen] = useState(false);
  const missing = missingChargeCode(diagnostic, source);
  if (!allowed || !missing) return null;
  return (
    <>
      <Button type="button" variant="outline" size="xs" onClick={() => setOpen(true)}>
        {t("Add {0} to accessorial charges", missing.code)}
      </Button>
      {open && (
        <AccessorialChargePanel
          open={open}
          onOpenChange={setOpen}
          mode="create"
          row={null}
          initial={missing}
          onCreated={() =>
            void queryClient.invalidateQueries({ queryKey: formulaTemplate.context.queryKey })
          }
        />
      )}
    </>
  );
}
