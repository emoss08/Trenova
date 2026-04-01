import { FormCreatePanel } from "@/components/form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { trailerSchema, type Trailer } from "@/types/trailer";
import { zodResolver } from "@hookform/resolvers/zod";
import { FileTextIcon } from "lucide-react";
import { lazy, useMemo } from "react";
import { useForm } from "react-hook-form";
import { TrailerForm } from "./trailer-form";

const DocumentsTab = lazy(() => import("@/components/documents/documents-tab"));

export function TrailerPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<Trailer>) {
  const form = useForm({
    resolver: zodResolver(trailerSchema),
    defaultValues: {
      status: "Available",
      code: "",
      model: "",
      make: "",
      year: undefined,
      licensePlateNumber: "",
      vin: "",
      registrationNumber: "",
      maxLoadWeight: undefined,
      lastInspectionDate: undefined,
      registrationExpiry: undefined,
      equipmentTypeId: "",
      equipmentManufacturerId: "",
      fleetCodeId: "",
      registrationStateId: "",
      createdAt: undefined,
      updatedAt: undefined,
      id: undefined,
      version: undefined,
      equipmentManufacturer: undefined,
      equipmentType: undefined,
      fleetCode: undefined,
      registrationState: undefined,
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
          resourceType: "trailer",
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
        url="/trailers/"
        queryKey="trailer-list"
        title="Trailer"
        fieldKey="code"
        formComponent={<TrailerForm />}
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
      url="/trailers/"
      queryKey="trailer-list"
      title="Trailer"
      formComponent={<TrailerForm />}
      useDock
    />
  );
}
