import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  hazardousMaterialSchema,
  HazardousMaterialSchema,
} from "@/lib/schemas/hazardous-material-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { HazardousMaterialForm } from "./hazardous-material-form";

export function EditHazardousMaterialModal({
  currentRecord,
}: EditTableSheetProps<HazardousMaterialSchema>) {
  const form = useForm({
    resolver: zodResolver(hazardousMaterialSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/hazardous-materials/"
      title="Hazardous Material"
      queryKey="hazardous-material-list"
      formComponent={<HazardousMaterialForm />}
      fieldKey="code"
      className="max-w-[550px]"
      form={form}
    />
  );
}
