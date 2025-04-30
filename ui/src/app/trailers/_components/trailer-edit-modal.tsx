import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  trailerSchema,
  type TrailerSchema,
} from "@/lib/schemas/trailer-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { TrailerForm } from "./trailer-form";

export function EditTrailerModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<TrailerSchema>) {
  const form = useForm({
    resolver: zodResolver(trailerSchema),
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
      className="max-w-[500px]"
    />
  );
}
