import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { tractorSchema, type Tractor } from "@/types/tractor";
import { zodResolver } from "@hookform/resolvers/zod";
import { FileTextIcon } from "lucide-react";
import { lazy, useMemo } from "react";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-form";

const DocumentsTab = lazy(() => import("@/components/documents/documents-tab"));

export function TractorPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<Tractor>) {
  const form = useForm({
    resolver: zodResolver(tractorSchema),
    defaultValues: {
      status: "Available",
      code: "",
      model: "",
      make: "",
      year: undefined,
      licensePlateNumber: "",
      vin: "",
      registrationNumber: "",
      registrationExpiry: undefined,
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
    },
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
        title="Tractor"
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
      title="Tractor"
      formComponent={<TractorForm />}
      useDock
    />
  );
}
