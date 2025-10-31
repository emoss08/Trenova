import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  fiscalYearSchema,
  FiscalYearSchema,
} from "@/lib/schemas/fiscal-year-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FiscalYearForm } from "./fiscal-year-form";

export function EditFiscalYearModal({
  currentRecord,
}: EditTableSheetProps<FiscalYearSchema>) {
  const form = useForm({
    resolver: zodResolver(fiscalYearSchema),
    defaultValues: currentRecord,
  });

  const {
    formState: { errors },
  } = form;
  console.log(errors);

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/fiscal-years/"
      title="Fiscal Year"
      queryKey="fiscal-year-list"
      formComponent={<FiscalYearForm />}
      fieldKey="year"
      form={form}
    />
  );
}
