import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { JSONEditorField } from "@/components/fields/sql-editor-field";
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
import { httpMethodChoices } from "@/lib/choices";
import {
  DataAPICallConfig,
  dataAPICallConfigSchema,
} from "@/lib/schemas/node-config-schema";
import { ActionConfigFormProps } from "@/types/workflow";
import { faBracketsSquare } from "@fortawesome/pro-regular-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus, X } from "lucide-react";
import { useCallback } from "react";
import {
  FormProvider,
  useFieldArray,
  useForm,
  useFormContext,
} from "react-hook-form";
import { VariableInput } from "./inputs/variable-input";
export function DataAPICallForm({
  initialConfig,
  onSave,
  onCancel,
}: Omit<ActionConfigFormProps, "actionType">) {
  const form = useForm<DataAPICallConfig>({
    resolver: zodResolver(dataAPICallConfigSchema),
    defaultValues: initialConfig,
  });

  const { control, handleSubmit, setError } = form;

  const handleSave = useCallback(
    (values: DataAPICallConfig) => {
      const validated = dataAPICallConfigSchema.safeParse(values);
      if (!validated.success) {
        validated.error.issues.forEach((issue) => {
          setError(issue.path[0] as keyof DataAPICallConfig, {
            message: issue.message,
          });
        });
        return;
      }

      onSave(validated.data);
    },
    [onSave, setError],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(handleSave)}>
        <FormGroup cols={1} className="px-4 py-2">
          <FormControl>
            <VariableInput
              name="url"
              control={control}
              rules={{ required: true }}
              label="URL"
              description="The URL of the API to call. Use {{trigger.url}} to reference the URL from the workflow trigger."
              placeholder="https://api.example.com/endpoint"
              type="url"
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="method"
              label="Method"
              placeholder="Method"
              description="The HTTP method to use for the API call."
              options={httpMethodChoices}
            />
          </FormControl>
          <HttpHeadersForm />
          <FormControl>
            <JSONEditorField
              control={control}
              name="body"
              label="Request Body (JSON)"
              description="The body of the request to send to the API. This should be a valid JSON object."
              placeholder='{"key": "value"}'
            />
          </FormControl>
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

function HttpHeadersForm() {
  const { control } = useFormContext<DataAPICallConfig>();

  const { fields, append, remove } = useFieldArray({
    control,
    name: "headers",
  });

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-col gap-0.5">
        <h3 className="text-sm font-medium">Headers</h3>
        <p className="text-xs text-muted-foreground">
          Add headers to the request to the API. You can use variables to insert
          values into the headers. For example, you can use{" "}
          {"{{trigger.customer.name}}"} to insert the customer name into the
          headers.
        </p>
      </div>
      {fields.length > 0 ? (
        <div className="flex flex-col gap-2">
          <ScrollArea className="flex max-h-64 rounded-md border border-dashed border-border">
            <div className="flex flex-col gap-2 p-4">
              {fields.map((field, index) => (
                <div
                  className="flex w-full items-center justify-between gap-2 md:flex-row"
                  key={field.id}
                >
                  <FormControl className="w-full">
                    <InputField
                      name={`headers.${index}.key`}
                      control={control}
                      rules={{ required: true }}
                      label="Key"
                      description="The key of the header."
                      placeholder="Key"
                    />
                  </FormControl>
                  <FormControl className="w-full">
                    <InputField
                      name={`headers.${index}.value`}
                      control={control}
                      rules={{ required: true }}
                      label="Value"
                      description="The value of the header."
                      placeholder="Value"
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
                      <TooltipContent>Remove header</TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </div>
              ))}
            </div>
          </ScrollArea>
          <Button
            type="button"
            variant="outline"
            onClick={() => append({ key: "", value: "" })}
            className="w-fit text-left"
          >
            <Plus className="size-3" />
            Add Header
          </Button>
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center gap-2 rounded-md border border-dashed border-border bg-sidebar p-2">
          <div className="flex items-center justify-center rounded-md bg-background p-2">
            <Icon icon={faBracketsSquare} className="size-10" />
          </div>
          <div className="flex flex-col items-center justify-center gap-2 text-center">
            <p className="bg-amber-300 px-1 py-0.5 text-center font-table text-sm/none font-medium text-amber-950 uppercase select-none dark:bg-amber-400 dark:text-neutral-900">
              No headers defined
            </p>

            <p className="bg-neutral-900 px-1 py-0.5 text-center font-table text-sm/none font-medium text-white uppercase select-none dark:bg-neutral-500 dark:text-neutral-900">
              Add headers to the request to the API.
            </p>
          </div>
          <Button
            type="button"
            variant="outline"
            onClick={() => append({ key: "", value: "" })}
            className="w-fit"
          >
            <Plus className="size-3" />
            Add Header
          </Button>
        </div>
      )}
    </div>
  );
}
