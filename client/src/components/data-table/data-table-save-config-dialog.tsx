"use no memo";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { apiService } from "@/services/api";
import {
  tableConfigurationFormSchema,
  type TableConfig,
  type TableConfiguration,
  type TableConfigurationFormValues,
} from "@/types/table-configuration";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { GlobeIcon, LockIcon } from "lucide-react";
import { useCallback } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";
import { SwitchField } from "../fields/switch-field";
import { TextareaField } from "../fields/textarea-field";

type DataTableSaveConfigDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  resource: string;
  currentConfig: TableConfig;
};

export function useVisibilityOptions() {
  return [
    {
      value: "Private",
      label: "Private",
      description: "Only you can see this view",
      icon: <LockIcon />,
    },
    {
      value: "Public",
      label: "Public",
      description: "Anyone in your organization can use this view",
      icon: <GlobeIcon />,
    },
  ];
}

export function DataTableSaveConfigDialog({
  open,
  onOpenChange,
  resource,
  currentConfig,
}: DataTableSaveConfigDialogProps) {
  const queryClient = useQueryClient();
  const visibilityOptions = useVisibilityOptions();

  const form = useForm<TableConfigurationFormValues>({
    resolver: zodResolver(tableConfigurationFormSchema),
    defaultValues: {
      name: "",
      description: "",
      resource,
      tableConfig: currentConfig,
      visibility: "Private",
      isDefault: false,
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    setError,
    setValue,
    formState: { isSubmitting },
  } = form;

  const { mutateAsync } = useApiMutation<
    TableConfiguration,
    TableConfigurationFormValues,
    unknown,
    TableConfigurationFormValues
  >({
    mutationFn: (data: TableConfigurationFormValues) =>
      apiService.tableConfigurationService.create(data),
    resourceName: "Table Configuration",
    setFormError: setError,
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["tableConfiguration"],
      });

      toast.success("Table Configuration created", {
        description: "Table Configuration created successfully",
      });
    },
  });

  const handleClose = useCallback(() => {
    onOpenChange(false);
    reset();
  }, [onOpenChange, reset]);

  const handleDialogOpenChange = useCallback(
    (nextOpen: boolean) => {
      if (nextOpen) {
        setValue("tableConfig", currentConfig);
        setValue("resource", resource);
      }
      onOpenChange(nextOpen);
    },
    [currentConfig, onOpenChange, resource, setValue],
  );

  const onSubmit = useCallback(
    async (values: TableConfigurationFormValues) => {
      await mutateAsync(values);
      handleClose();
    },
    [mutateAsync, handleClose],
  );

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogContent className="max-w-[400px]">
        <DialogHeader>
          <DialogTitle>Save View</DialogTitle>
          <DialogDescription>
            Save the current table configuration for quick access later.
          </DialogDescription>
        </DialogHeader>
        <Form onSubmit={handleSubmit(onSubmit)}>
          <FormGroup cols={1} className="pb-4">
            <FormControl cols="full">
              <InputField
                name="name"
                control={control}
                label="Name"
                placeholder="e.g., Active shipments this week"
                description="A descriptive name for this table configuration"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl cols="full">
              <TextareaField
                name="description"
                control={control}
                label="Description"
                placeholder="The description of the table configuration."
                description="The description of the table configuration."
                rules={{ required: false }}
              />
            </FormControl>
            <FormControl cols="full">
              <SelectField
                name="visibility"
                control={control}
                label="Visibility"
                placeholder="Select visibility"
                description="The visibility of the table configuration."
                options={visibilityOptions}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl cols="full">
              <SwitchField
                outlined
                position="left"
                name="isDefault"
                control={control}
                label="Set as default"
                description="When enabled, the system will automatically apply this table configuration to the table"
                rules={{ required: false }}
              />
            </FormControl>
          </FormGroup>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              isLoading={isSubmitting}
              loadingText="Saving..."
            >
              Save View
            </Button>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}

export default DataTableSaveConfigDialog;
