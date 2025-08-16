import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  holdReasonSchema,
  HoldReasonSchema,
} from "@/lib/schemas/hold-reason-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { HoldReasonForm } from "./hold-reason-form";

export function EditHoldReasonModal({
  currentRecord,
}: EditTableSheetProps<HoldReasonSchema>) {
  const form = useForm({
    resolver: zodResolver(holdReasonSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/hold-reasons/"
      title="Hold Reason"
      queryKey="hold-reason-list"
      formComponent={<HoldReasonForm />}
      fieldKey="code"
      form={form}
    />
  );
}
