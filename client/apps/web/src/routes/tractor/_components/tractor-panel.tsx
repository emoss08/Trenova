import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import type { TractorRow } from "@/lib/graphql/equipment-table";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { tractorSchema } from "@/types/tractor";
import { zodResolver } from "@hookform/resolvers/zod";
import { ClipboardCheckIcon, FileTextIcon } from "lucide-react";
import { lazy, useMemo } from "react";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-form";

const DocumentsTab = lazy(() => import("@/components/documents/documents-tab"));
const InspectionsTab = lazy(() => import("./tractor-inspections-tab"));

export function buildTractorDefaults() {
  return {
    status: "Available" as const,
    code: "",
    model: "",
    make: "",
    year: undefined,
    licensePlateNumber: "",
    vin: "",
    registrationNumber: "",
    registrationExpiry: undefined,
    externalId: "",
    fuelType: "Diesel" as const,
    iftaQualified: true,
    equipmentTypeId: "",
    equipmentManufacturerId: "",
    fleetCodeId: "",
    stateId: "",
    primaryWorkerId: "",
    secondaryWorkerId: "",
    createdAt: undefined,
    updatedAt: undefined,
    id: undefined,
    version: undefined,
    equipmentManufacturer: undefined,
    equipmentType: undefined,
    fleetCode: undefined,
    state: undefined,
    primaryWorker: undefined,
    secondaryWorker: undefined,
  };
}

export function TractorPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<TractorRow>) {
  const t = useT();

  const form = useForm({
    resolver: zodResolver(tractorSchema),
    defaultValues: buildTractorDefaults(),
  });

  const documentsTabs = useMemo(
    () => [
      {
        value: "documents",
        label: "Documents",
        icon: FileTextIcon,
        content: DocumentsTab,
        contentProps: {
          resourceType: "tractor",
          resourceId: row?.id,
        },
      },
      {
        value: "inspections",
        label: "Inspections",
        icon: ClipboardCheckIcon,
        content: InspectionsTab,
        contentProps: {
          tractorId: row?.id,
        },
      },
    ],
    [row?.id],
  );

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/tractors/"
        queryKey="tractor-list"
        title={t("Tractor")}
        fieldKey="code"
        formComponent={<TractorForm />}
        tabs={documentsTabs}
        useDock
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/tractors/"
      queryKey="tractor-list"
      title={t("Tractor")}
      formComponent={<TractorForm />}
      useDock
    />
  );
}
