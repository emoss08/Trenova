import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  hazardousMaterialSchema,
  HazardousMaterialSchema,
} from "@/lib/schemas/hazardous-material-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { HazardousMaterialForm } from "./hazardous-material-form";

export function EditHazardousMaterialModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<HazardousMaterialSchema>) {
  const form = useForm<HazardousMaterialSchema>({
    resolver: yupResolver(hazardousMaterialSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/hazardous-materials/"
      title="Hazardous Material"
      queryKey={["hazardous-material"]}
      formComponent={<HazardousMaterialForm />}
      fieldKey="code"
      className="max-w-[550px]"
      form={form}
      schema={hazardousMaterialSchema}
    />
  );
}
