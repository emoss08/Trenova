import { TabbedFormCreatePanel } from "@/components/tabbed-form-create-panel";
import { TabbedFormEditPanel } from "@/components/tabbed-form-edit-panel";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { rateMatrixSchema, type RateMatrix } from "@trenova/shared/types/rate";
import { FileTextIcon, Grid3x3Icon, TableIcon } from "lucide-react";
import { useMemo } from "react";
import { useForm, type Resolver } from "react-hook-form";
import { DimensionEditor } from "./dimension-editor";
import { MatrixGridEditor } from "./matrix-grid-editor";
import { RateMatrixForm } from "./rate-matrix-form";

const DEFAULT_MATRIX: Partial<RateMatrix> = {
  code: "",
  name: "",
  description: "",
  status: "Active",
  formulaTemplateId: "",
  currency: "USD",
  roundingMode: "HalfUp",
  roundingPrecision: 2,
  dimensions: [],
};

export function RateMatrixPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<RateMatrix>) {
  const form = useForm<RateMatrix>({
    resolver: zodResolver(rateMatrixSchema) as Resolver<RateMatrix>,
    defaultValues: DEFAULT_MATRIX as RateMatrix,
    mode: "onChange",
  });

  const formTabs = useMemo(
    () => [
      {
        value: "overview",
        label: "Overview",
        icon: FileTextIcon,
        content: <RateMatrixForm />,
      },
      {
        value: "axes",
        label: "Axes",
        icon: Grid3x3Icon,
        content: <DimensionEditor />,
      },
      {
        value: "rates",
        label: "Rates",
        icon: TableIcon,
        content: <MatrixGridEditor rateMatrixId={row?.id} />,
      },
    ],
    [row?.id],
  );

  if (mode === "edit") {
    return (
      <TabbedFormEditPanel<RateMatrix, RateMatrix>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        size="xl"
        queryKey="rate-matrix-list"
        title="Rate Matrix"
        fieldKey="name"
        formTabs={formTabs}
        mutationFn={(values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No Rate Matrix ID selected");
          }

          return apiService.rateMatrixService.update(currentRow.id, values);
        }}
      />
    );
  }

  return (
    <TabbedFormCreatePanel<RateMatrix, RateMatrix>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      size="xl"
      queryKey="rate-matrix-list"
      title="Rate Matrix"
      description="Enter a published tariff as the grid it was published as, and point any lane at it."
      formTabs={formTabs}
      mutationFn={(values) => apiService.rateMatrixService.create(values)}
    />
  );
}
