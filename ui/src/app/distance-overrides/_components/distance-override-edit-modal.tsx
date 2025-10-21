import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  distanceOverrideSchema,
  type DistanceOverrideSchema,
} from "@/lib/schemas/distance-override-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DistanceOverrideForm } from "./distance-override-form";

export function EditDistanceOverrideModal({
  currentRecord,
}: EditTableSheetProps<DistanceOverrideSchema>) {
  const form = useForm({
    resolver: zodResolver(distanceOverrideSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/distance-overrides/"
      title="Distance Override"
      queryKey="distance-override-list"
      formComponent={<DistanceOverrideForm />}
      form={form}
      className="sm:max-w-[500px]"
    />
  );
}
