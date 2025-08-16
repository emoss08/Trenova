import { FormCreateModal } from "@/components/ui/form-create-modal";
import { holdReasonSchema } from "@/lib/schemas/hold-reason-schema";
import { HoldSeverity, HoldType } from "@/lib/schemas/shipment-hold-schema";
import { TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { HoldReasonForm } from "./hold-reason-form";

export function CreateHoldReasonModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(holdReasonSchema),
    defaultValues: {
      active: true,
      type: HoldType.enum.OperationalHold,
      code: "",
      label: "",
      description: "",
      defaultSeverity: HoldSeverity.enum.Advisory,
      defaultBlocksDispatch: false,
      defaultBlocksDelivery: false,
      defaultBlocksBilling: false,
      defaultVisibleToCustomer: false,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Hold Reason"
      formComponent={<HoldReasonForm />}
      form={form}
      url="/hold-reasons/"
      queryKey="hold-reason-list"
    />
  );
}
