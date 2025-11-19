import { InputField } from "@/components/fields/input-field";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  DocumentValidateCompletenessConfig,
  documentValidateCompletenessConfigSchema,
} from "@/lib/schemas/node-config-schema";
import { ActionConfigFormProps } from "@/types/workflow";
import { faFile } from "@fortawesome/pro-regular-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus, X } from "lucide-react";
import {
  FormProvider,
  useFieldArray,
  useForm,
  useFormContext,
} from "react-hook-form";
import { VariableInput } from "./inputs/variable-input";

export function DocumentValidateCompletenessForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const form = useForm<DocumentValidateCompletenessConfig>({
    resolver: zodResolver(documentValidateCompletenessConfigSchema),
    defaultValues: initialConfig,
  });
  const { control, handleSubmit } = form;

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSave)}>
        <FormGroup className="px-4 pb-2">
          <FormControl>
            <VariableInput
              name="shipmentId"
              control={control}
              rules={{ required: true }}
              label="Shipment ID"
              description="The ID of the shipment to run document validation against"
              placeholder="{{ trigger.shipmentId }}"
              type="text"
            />
          </FormControl>
          <RequiredDocumentsForm />
        </FormGroup>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button type="submit">Save Configuration</Button>
        </DialogFooter>
      </Form>
    </FormProvider>
  );
}

function RequiredDocumentsForm() {
  const { control } = useFormContext<DocumentValidateCompletenessConfig>();

  const { fields, append, remove } = useFieldArray({
    control,
    name: "requiredDocuments",
  });

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-0.5">
        <h3 className="text-sm font-medium">Required Documents</h3>
        <p className="text-xs text-muted-foreground">
          Add the types of documents that are required for the shipment. You can
          use variables to insert values into the document types. For example,
          you can use {"{{trigger.customer.name}}"} to insert the customer name
          into the document types.
        </p>
      </div>
      {fields.length > 0 ? (
        <div className="flex flex-col gap-2">
          <ScrollArea className="flex max-h-64 rounded-md border border-dashed border-border">
            <div className="flex flex-col gap-2 p-4">
              {fields.map((field, index) => (
                <div
                  key={field.id}
                  className="flex w-full items-center justify-between gap-2 md:flex-row"
                >
                  <FormControl className="w-full">
                    <InputField
                      name={`requiredDocuments.${index}.value`}
                      control={control}
                      rules={{ required: true }}
                      label="Document Type"
                      description="The type of the document to validate."
                      placeholder="Document type (e.g., BOL, POD, Invoice)"
                    />
                  </FormControl>
                  <TooltipProvider>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          type="button"
                          variant="outline"
                          size="icon"
                          className="mt-1 size-7 shrink-0"
                          onClick={() => remove(index)}
                        >
                          <X className="size-3" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>Remove required document</TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </div>
              ))}
            </div>
          </ScrollArea>
          <Button
            type="button"
            variant="outline"
            onClick={() => append({ value: "" })}
            className="w-fit"
          >
            <Plus className="size-3" />
            Add Required Document
          </Button>
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed border-border bg-sidebar p-2">
          <div className="flex items-center justify-center rounded-md bg-background p-2">
            <Icon icon={faFile} className="size-10" />
          </div>
          <div className="flex flex-col items-center justify-center gap-2 text-center">
            <p className="bg-amber-300 px-1 py-0.5 text-center font-table text-sm/none font-medium text-amber-950 uppercase select-none dark:bg-amber-400 dark:text-neutral-900">
              No required documents defined
            </p>

            <p className="bg-neutral-900 px-1 py-0.5 text-center font-table text-sm/none font-medium text-white uppercase select-none dark:bg-neutral-500 dark:text-neutral-900">
              Add required documents to the shipment.
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={() => append({ value: "" })}
            className="w-fit"
          >
            <Plus className="size-3" />
            Add Required Document
          </Button>
        </div>
      )}
    </div>
  );
}
