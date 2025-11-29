import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { DocumentTypeAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  documentTemplateOrientationChoices,
  documentTemplatePageSizeChoices,
  documentTemplateStatusChoices,
} from "@/lib/choices";
import { DocumentTemplateSchema } from "@/lib/schemas/document-template-schema";
import { FileText, Ruler, Settings, Settings2, Type } from "lucide-react";
import React from "react";
import { useFormContext } from "react-hook-form";
import { CollapsibleSection } from "../collapsible-section";

export function DocumentTemplateSettingsForm() {
  return (
    <DocumentTemplateSettingsOuter>
      <DocumentTemplateSettingsHeader />
      <DocumentTemplateSettingsContent />
    </DocumentTemplateSettingsOuter>
  );
}

function DocumentTemplateSettingsOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex h-full flex-col">{children}</div>;
}

function DocumentTemplateSettingsHeader() {
  return (
    <div className="flex h-[55px] shrink-0 items-center gap-2 border-b border-border bg-gradient-to-r from-primary/5 to-transparent p-3">
      <div className="flex size-8 items-center justify-center rounded-lg bg-primary/10">
        <Settings2 className="size-4 text-primary" />
      </div>
      <div className="min-w-0 flex-1">
        <h3 className="truncate text-sm font-semibold">Template Settings</h3>
        <p className="truncate text-2xs text-muted-foreground">
          Configure your document
        </p>
      </div>
    </div>
  );
}

function DocumentTemplateSettingsContent() {
  const { control } = useFormContext<DocumentTemplateSchema>();

  return (
    <ScrollArea className="flex max-h-[calc(100vh-15rem)] flex-1 flex-col p-4">
      <div className="space-y-3">
        <CollapsibleSection title="Basic Information" icon={FileText}>
          <FormGroup cols={1}>
            <FormControl>
              <InputField
                control={control}
                name="code"
                label="Template Code"
                placeholder="INV-TPL-001"
                rules={{ required: "Code is required" }}
                maxLength={50}
              />
            </FormControl>
            <FormControl>
              <InputField
                control={control}
                name="name"
                label="Template Name"
                placeholder="Standard Invoice"
                rules={{ required: "Name is required" }}
                maxLength={200}
              />
            </FormControl>
            <FormControl>
              <TextareaField
                control={control}
                name="description"
                label="Description"
                placeholder="Describe what this template is used for..."
                className="min-h-[60px]"
              />
            </FormControl>
            <FormControl>
              <DocumentTypeAutocompleteField<DocumentTemplateSchema>
                control={control}
                name="documentTypeId"
                label="Document Type"
                rules={{ required: "Document type is required" }}
                placeholder="Select document type"
                description="The document type that this template is associated with."
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="status"
                label="Status"
                options={documentTemplateStatusChoices}
                rules={{ required: "Status is required" }}
              />
            </FormControl>
          </FormGroup>
        </CollapsibleSection>
        <CollapsibleSection
          title="Page Configuration"
          icon={Type}
          defaultOpen={false}
        >
          <FormGroup cols={1}>
            <FormControl>
              <SelectField
                control={control}
                name="pageSize"
                label="Page Size"
                options={documentTemplatePageSizeChoices}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name="orientation"
                label="Orientation"
                options={documentTemplateOrientationChoices}
              />
            </FormControl>
          </FormGroup>
        </CollapsibleSection>
        <CollapsibleSection
          title="Page Margins"
          icon={Ruler}
          defaultOpen={false}
        >
          <div className="space-y-3">
            <p className="text-2xs text-muted-foreground">
              Set margins in millimeters
            </p>
            <FormGroup cols={2}>
              <FormControl>
                <NumberField
                  control={control}
                  name="marginTop"
                  label="Top"
                  min={0}
                  max={100}
                  sideText="mm"
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="marginBottom"
                  label="Bottom"
                  min={0}
                  max={100}
                  sideText="mm"
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="marginLeft"
                  label="Left"
                  min={0}
                  max={100}
                  sideText="mm"
                />
              </FormControl>
              <FormControl>
                <NumberField
                  control={control}
                  name="marginRight"
                  label="Right"
                  min={0}
                  max={100}
                  sideText="mm"
                />
              </FormControl>
            </FormGroup>
          </div>
        </CollapsibleSection>
        <CollapsibleSection title="Options" icon={Settings} defaultOpen={false}>
          <SwitchField
            control={control}
            name="isDefault"
            label="Default Template"
            description="Use this template by default for the selected document type"
            outlined
          />
        </CollapsibleSection>
      </div>
    </ScrollArea>
  );
}
