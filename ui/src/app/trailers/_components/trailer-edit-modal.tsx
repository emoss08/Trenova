import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  trailerSchema,
  type TrailerSchema,
} from "@/lib/schemas/trailer-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { TrailerForm } from "./trailer-form";

export function EditTrailerModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<TrailerSchema>) {
  const form = useForm<TrailerSchema>({
    resolver: yupResolver(trailerSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/trailers/"
      title="Trailer"
      queryKey="trailer-list"
      formComponent={<TrailerForm />}
      fieldKey="code"
      form={form}
      schema={trailerSchema}
      className="max-w-[500px]"
    />
  );
}
