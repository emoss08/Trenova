import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  tableConfigurationSchema,
  type TableConfigurationSchema,
} from "@/lib/schemas/table-configuration-schema";
import type { Resource } from "@/types/audit-entry";
import { TableSheetProps } from "@/types/data-table";
import { Visibility } from "@/types/table-configuration";
import { zodResolver } from "@hookform/resolvers/zod";
import type { VisibilityState } from "@tanstack/react-table";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { TableConfigurationForm } from "./table-configuration-form";

type CreateTableConfigurationModalProps = TableSheetProps & {
  resource: Resource;
  visiblityState: VisibilityState;
};

export function CreateTableConfigurationModal({
  open,
  onOpenChange,
  resource,
  visiblityState,
}: CreateTableConfigurationModalProps) {
  const form = useForm<TableConfigurationSchema>({
    resolver: zodResolver(tableConfigurationSchema),
    defaultValues: {
      name: "",
      description: "",
      visibility: Visibility.Private,
      isDefault: false,
      resource: resource,
      tableConfig: {
        columnVisibility: visiblityState,
      },
    },
  });

  // Ensure tableConfig is registered so it is included in submitted data
  useEffect(() => {
    // RHF requires fields to be registered; register nested object manually
    form.register("tableConfig");
  }, [form]);

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Table Configuration"
      formComponent={<TableConfigurationForm />}
      form={form}
      url="/table-configurations/"
      queryKey="table-configurations"
    />
  );
}
