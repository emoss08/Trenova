import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import type { ChecklistKind } from "@/types/case-checklist";
import { QueryLazyComponent } from "@trenova/shared/components/error-boundary";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { lazy, useState } from "react";

const ChecklistWorkspace = lazy(() => import("./_components/checklist-workspace"));

/**
 * How Desk case checklists are laid out: for a shipment, what stands
 * between it and the bill; for an invoice, between it and closing. The
 * organization sets the steps, their order and which are required, and a
 * customer that bills differently can have its own.
 */
export function CaseChecklistsPage() {
  const t = useT();
  const [kind, setKind] = useState<ChecklistKind>("ReadyToBill");

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Case checklists"),
        description: t(
          "The steps a Desk case shows between a record and what comes next, for your organization and for each customer that bills differently.",
        ),
      }}
    >
      <SegmentedControl<ChecklistKind>
        aria-label={t("Checklist")}
        className="w-fit"
        value={kind}
        onValueChange={setKind}
        items={[
          { value: "ReadyToBill", label: t("Ready to bill"), caption: t("Shipments") },
          { value: "ReadyToClose", label: t("Ready to close"), caption: t("Invoices") },
        ]}
      />
      <QueryLazyComponent queryKey={queries.caseChecklist.list._def}>
        <ChecklistWorkspace key={kind} kind={kind} />
      </QueryLazyComponent>
    </PageLayout>
  );
}
