import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DocumentPacketRuleRow } from "@/lib/graphql/document-packet-rule-table";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import {
  createDocumentPacketRule,
  updateDocumentPacketRule,
} from "@/services/document-packet-rule";
import { documentPacketRuleSchema, type DocumentPacketRule } from "@/types/document-packet-rule";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DocumentPacketRuleForm } from "./document-packet-rule-form";

export function DocumentPacketRulePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<DocumentPacketRuleRow>) {
  const t = useT();

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
        title={t("Document packet rule")}
        queryKey="document-packet-rule-list"
        formComponent={<DocumentPacketRuleForm />}
        form={form}
        mutationFn={async (values, current) => {
          await updateDocumentPacketRule(current.id, { ...values, version: current.version });
          return values;
        }}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      mutationFn={async (values) => {
        await createDocumentPacketRule(values);
        return values;
      }}
      queryKey="document-packet-rule-list"
      title={t("Document packet rule")}
      formComponent={<DocumentPacketRuleForm />}
    />
  );
}
