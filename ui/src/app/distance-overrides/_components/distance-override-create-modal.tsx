import { FormCreateModal } from "@/components/ui/form-create-modal";
import { distanceOverrideSchema } from "@/lib/schemas/distance-override-schema";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DistanceOverrideForm } from "./distance-override-form";

export function CreateDistanceOverrideModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(distanceOverrideSchema),
    defaultValues: {
      createdAt: undefined,
      updatedAt: undefined,
      id: undefined,
      version: undefined,
      originLocationId: "",
      destinationLocationId: "",
      customerId: "",
      distance: undefined,
      customer: undefined,
      destinationLocation: undefined,
      originLocation: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Distance Override"
      formComponent={<DistanceOverrideForm />}
      form={form}
      url="/distance-overrides/"
      queryKey="distance-override-list"
      className="sm:max-w-[500px]"
    />
  );
}
