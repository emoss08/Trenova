import { FormCreateModal } from "@/components/ui/form-create-modal";
import { getEndOfMonth, getStartOfMonth } from "@/lib/date";
import {
  fiscalPeriodSchema,
  FiscalPeriodStatusSchema,
} from "@/lib/schemas/fiscal-period-schema";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FiscalPeriodForm } from "./fiscal-period-form";

export function CreateFiscalPeriodModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const firstOfMonth = getStartOfMonth(new Date());
  const endOfMonth = getEndOfMonth(new Date());

  const form = useForm({
    resolver: zodResolver(fiscalPeriodSchema),
    defaultValues: {
      status: FiscalPeriodStatusSchema.enum.Open,
      name: "",
      startDate: firstOfMonth,
      endDate: endOfMonth,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Fiscal Period"
      formComponent={<FiscalPeriodForm isCreate />}
      description="Add a new fiscal period for historical data or future planning."
      form={form}
      url="/fiscal-periods/"
      queryKey="fiscal-period-list"
    />
  );
}
