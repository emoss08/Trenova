import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  fiscalPeriodSchema,
  FiscalPeriodSchema,
} from "@/lib/schemas/fiscal-period-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FiscalPeriodForm } from "./fiscal-period-form";

export function EditFiscalPeriodModal({
  currentRecord,
}: EditTableSheetProps<FiscalPeriodSchema>) {
  const form = useForm({
    resolver: zodResolver(fiscalPeriodSchema),
    defaultValues: currentRecord,
  });

  const {
    formState: { errors },
  } = form;
  console.log(errors);

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/fiscal-periods/"
      title="Fiscal Period"
      queryKey="fiscal-period-list"
      formComponent={<FiscalPeriodForm />}
      fieldKey="name"
      form={form}
    />
  );
}
