/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */



import { Label } from "@/components/common/fields/label";
import {
  RadioGroup,
  RadioGroupItem,
} from "@/components/common/fields/radio-group";
import { SelectInput } from "@/components/common/fields/select-input";
import { TextareaField } from "@/components/common/fields/textarea";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useReportColumns } from "@/hooks/useQueries";
import axios from "@/lib/axiosConfig";
import { StoreType } from "@/lib/useGlobalStore";
import { ExportModelSchema } from "@/lib/validations/GenericSchema";
import { TableStoreProps, useTableStore as store } from "@/stores/TableStore";
import { IChoiceProps } from "@/types";
import { DeliveryMethodChoices, TExportModelFormValues } from "@/types/forms";
import { faDownload, faEnvelope } from "@fortawesome/pro-regular-svg-icons";
import { yupResolver } from "@hookform/resolvers/yup";
import { DialogTitle } from "@radix-ui/react-dialog";
import { EllipsisVerticalIcon } from "lucide-react";
import React, { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

interface Props {
  store: StoreType<TableStoreProps>;
  modelName: string;
  name: string;
}

const deliveryMethodChoices = [
  {
    label: "Local",
    value: "local",
    color: "#15803d",
    description: "Save to your local instance",
    icon: faDownload,
  },
  {
    label: "Email",
    value: "email",
    color: "#2563eb",
    description: "Send via email",
    icon: faEnvelope,
  },
] satisfies ReadonlyArray<IChoiceProps<DeliveryMethodChoices>>;

function TableExportModalBody({
  modelName,
  showExportModal,
  setShowExportModal,
}: {
  modelName: string;
  showExportModal: boolean;
  setShowExportModal: React.Dispatch<React.SetStateAction<boolean>>;
}) {
  const [loading, setLoading] = React.useState<boolean>(false);
  const [selectedColumns, setSelectedColumns] = React.useState<string[]>([]);
  const [showEmailField, setShowEmailField] = React.useState<boolean>(false);
  const { groupedOptions, isError, isLoading } = useReportColumns(
    modelName,
    showExportModal,
  );

  const { control, handleSubmit, reset, watch, setError } =
    useForm<TExportModelFormValues>({
      resolver: yupResolver(ExportModelSchema),
      defaultValues: {
        columns: [],
        deliveryMethod: "local",
        fileFormat: "csv",
        emailRecipients: undefined,
      },
    });

  useEffect(() => {
    const subscription = watch((value, { name }) => {
      // Set selected columns
      if (name === "columns" && value.columns) {
        setSelectedColumns(value.columns as string[]);
      }

      // Show email field if delivery method is email
      if (name === "deliveryMethod" && value.deliveryMethod === "email") {
        setShowEmailField(true);
      } else if (
        name === "deliveryMethod" &&
        value.deliveryMethod === "local"
      ) {
        setShowEmailField(false);
      }
    });

    return () => subscription.unsubscribe();
  }, [setSelectedColumns, watch]);

  const processColumnsAndRelationships = (columns: string[]) => {
    const mainColumns: any = [];
    const relationships: any = {};

    columns.forEach((column) => {
      const parts = column.split(".");
      if (parts.length === 1) {
        // Main table column
        mainColumns.push(column);
      } else {
        // Related table column
        const [foreignKey, referencedTable, referencedColumn] = parts;
        if (!relationships[foreignKey]) {
          relationships[foreignKey] = {
            foreignKey,
            referencedTable,
            columns: [],
          };
        }
        relationships[foreignKey].columns.push(referencedColumn);
      }
    });

    return { mainColumns, relationships: Object.values(relationships) };
  };

  const submitForm = async (values: TExportModelFormValues) => {
    setLoading(true);

    const { mainColumns, relationships } = processColumnsAndRelationships(
      values.columns,
    );

    try {
      const response = await axios.post("reports/generate/", {
        tableName: modelName as string,
        fileFormat: values.fileFormat,
        columns: mainColumns,
        relationships: relationships,
        deliveryMethod: values.deliveryMethod,
        emailRecipients: values.emailRecipients,
      });

      if (response.status === 200) {
        setShowExportModal(false);
        toast.success(
          <div className="flex flex-col space-y-1">
            <span className="font-semibold">Export job sent!</span>
            <span className="text-xs">
              Export job has been sent. You will receive a notification once
              it's ready.
            </span>
          </div>,
        );
        reset();
      }
    } catch (error: any) {
      if (error.response && error.response.data) {
        const { data } = error.response;
        Object.entries(data).forEach(([key, value]) => {
          // Check if the value is an object and has a property 'message'
          if (
            typeof value === "object" &&
            value !== null &&
            "message" in value
          ) {
            // If so, use the 'message' for setting the error
            setError(key as any, {
              type: "manual",
              message: value.message as string,
            });
          } else {
            // If it's not an object or doesn't have a 'message', set the value directly
            setError(key as any, {
              type: "manual",
              message: value as string,
            });
          }
        });
      }

      toast.error(
        <div className="flex flex-col space-y-1">
          <span className="font-semibold">
            {error.response.data.code || "Error"}
          </span>
          <span className="text-xs">
            {error.response.data.message ||
              "An error occurred, check the form and try again."}
          </span>
        </div>,
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(submitForm)}>
      <div className="mb-5">
        <SelectInput
          isMulti
          hideSelectedOptions={true}
          control={control}
          rules={{ required: true }}
          isLoading={isLoading}
          isFetchError={isError}
          name="columns"
          options={groupedOptions}
          label="Columns"
          placeholder="Select columns"
          description="A group of columns/fields that will be exported into your specified format."
        />
      </div>
      <div className="mb-5">
        <SelectInput
          control={control}
          rules={{ required: true }}
          name="deliveryMethod"
          options={deliveryMethodChoices}
          label="Delivery Method"
          placeholder="Select delivery method"
          description="Select a delivery method for the export. You can either download the file or receive it via email."
        />
      </div>
      <div className="mb-5">
        <Label className="required">Export Format</Label>
        <Controller
          name="fileFormat"
          control={control}
          defaultValue="csv"
          render={({ field: { onChange, value } }) => (
            <RadioGroup
              className="mt-1 grid grid-cols-3"
              onValueChange={onChange}
              defaultValue={value}
            >
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="csv" id="r1" />
                <Label htmlFor="r1">CSV</Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="xlsx" id="r2" />
                <Label htmlFor="r2">Excel</Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem disabled value="pdf" id="r3" />
                <Label htmlFor="r3">PDF</Label>
              </div>
            </RadioGroup>
          )}
        />
        <p className="text-foreground/70 mt-1 text-xs">
          Select a format to export (CSV, Excel, or PDF).
        </p>
      </div>
      {showEmailField && (
        <div className="mb-5">
          <TextareaField
            control={control}
            rules={{ required: true }}
            name="emailRecipients"
            label="Email Recipients"
            placeholder="Enter Email Recipients"
            description="Enter the email addresses of the recipients. You can enter multiple email addresses separated by a comma."
          />
        </div>
      )}
      <div className="mt-5 flex justify-end gap-4 border-t pt-2">
        <Button
          type="button"
          variant="outline"
          onClick={() => setShowExportModal(false)}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          isLoading={loading}
          loadingText="Sending Job..."
          disabled={selectedColumns?.length === 0}
        >
          Export
        </Button>
      </div>
    </form>
  );
}

export function TableExportModal({ store, modelName, name }: Props) {
  const [showExportModal, setShowExportModal] = store.use("exportModalOpen");

  if (!setShowExportModal) return null;

  return (
    <Dialog open={showExportModal} onOpenChange={setShowExportModal}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Export {name}</DialogTitle>
        </DialogHeader>
        <TableExportModalBody
          showExportModal={showExportModal}
          modelName={modelName}
          setShowExportModal={setShowExportModal}
        />
      </DialogContent>
    </Dialog>
  );
}

export function DataTableImportExportOption() {
  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button size="sm" variant="outline" className="h-8 lg:flex">
            <EllipsisVerticalIcon className="mr-1 mt-0.5 size-4" />
            Options
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-[150px]">
          <DropdownMenuLabel>Options</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem>Import</DropdownMenuItem>
          <DropdownMenuItem onClick={() => store.set("exportModalOpen", true)}>
            Export
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem>View Audit Log</DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
