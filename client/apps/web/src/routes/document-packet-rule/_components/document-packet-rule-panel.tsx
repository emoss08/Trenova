import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { documentPacketRuleSchema, type DocumentPacketRule } from "@/types/document-packet-rule";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DocumentPacketRuleForm } from "./document-packet-rule-form";

export function DocumentPacketRulePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<DocumentPacketRule>) {
  const form = useForm<DocumentPacketRule>({
    resolver: zodResolver(documentPacketRuleSchema),
    defaultValues: {
      resourceType: "Shipment",
      documentTypeId: "",
      required: false,
      allowMultiple: false,
      displayOrder: 0,
      expirationRequired: false,
      expirationWarningDays: 30,
    },
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        url="/document-packet-rules/"
        title="Document Packet Rule"
        queryKey="document-packet-rule-list"
        formComponent={<DocumentPacketRuleForm />}
        form={form}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/document-packet-rules/"
      queryKey="document-packet-rule-list"
      title="Document Packet Rule"
      formComponent={<DocumentPacketRuleForm />}
    />
  );
}
