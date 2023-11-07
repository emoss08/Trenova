/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import React from "react";
import { useQuery } from "@tanstack/react-query";
import { TExportModelFormValues } from "@/types/forms";
import { getColumns } from "@/services/ReportRequestService";
import axios from "@/lib/axiosConfig";
import { ExportModelSchema } from "@/lib/validations/GenericSchema";
import { Controller, useForm } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import { toast } from "@/components/ui/use-toast";
import { Dialog, DialogContent, DialogHeader } from "@/components/ui/dialog";
import { DialogTitle } from "@radix-ui/react-dialog";
import { SelectInput } from "@/components/common/fields/select-input";
import {
  RadioGroup,
  RadioGroupItem,
} from "@/components/common/fields/radio-group";
import { Label } from "@/components/common/fields/label";
import { Button } from "@/components/ui/button";
import { StoreType } from "@/lib/useGlobalStore";
import { TableStoreProps, useTableStore as store } from "@/stores/TableStore";
import { Loader2 } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { DotsVerticalIcon } from "@radix-ui/react-icons";

interface Props {
  store: StoreType<TableStoreProps>;
  modelName: string;
  name: string;
}

function TableExportModalBody({
  name,
  modelName,
  showExportModal,
  setShowExportModal,
}: {
  name: string;
  modelName: string;
  showExportModal: boolean;
  setShowExportModal: React.Dispatch<React.SetStateAction<boolean>>;
}) {
  const [loading, setLoading] = React.useState<boolean>(false);

  const { data: columnsData, isLoading: isColumnsLoading } = useQuery({
    queryKey: [`${modelName}-Columns`],
    queryFn: () => getColumns(modelName as string),
    enabled: showExportModal,
    staleTime: Infinity,
  });

  const { control, handleSubmit, reset, watch } =
    useForm<TExportModelFormValues>({
      resolver: yupResolver(ExportModelSchema),
      defaultValues: {
        columns: [],
        fileFormat: "csv",
      },
    });

  const watchedColumns = watch("columns");

  const columns = columnsData?.map((column: any) => ({
    label: column.label,
    value: column.value,
  }));

  const submitForm = async (values: TExportModelFormValues) => {
    setLoading(true);

    try {
      const response = await axios.post("generate_report/", {
        modelName: modelName as string,
        fileFormat: values.fileFormat,
        columns: values.columns,
      });

      if (response.status === 202) {
        setShowExportModal(false);
        toast({
          title: "Success",
          description: response.data.results,
        });
        reset();
      }
    } catch (error: any) {
      toast({
        title: "Error",
        description: error.response.data.error,
      });
    } finally {
      setLoading(false);
    }
  };

  return isColumnsLoading ? (
    <>
      <div className="flex flex-col items-center justify-center space-y-2 h-40 w-full">
        <Loader2 className="animate-spin h-20 w-20 text-muted" />
        <p className="text-center">
          Fetching columns for {name.toLowerCase()}s...
        </p>
      </div>
    </>
  ) : (
    <form onSubmit={handleSubmit(submitForm)}>
      <div className="mb-5">
        <SelectInput
          isMulti
          hideSelectedOptions={true}
          control={control}
          rules={{ required: true }}
          name="columns"
          options={columns}
          label="Columns"
          placeholder="Select columns"
          description="Fields with underscores are related fields. For example, 'organization__name' is the 'name' field of the organization of the record."
        />
      </div>
      <div>
        <Label className="required">Export Format</Label>
        <Controller
          name="fileFormat"
          control={control}
          defaultValue="csv"
          render={({ field: { onChange, value } }) => (
            <RadioGroup
              className="grid grid-cols-3 mt-1"
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
                <RadioGroupItem value="pdf" id="r3" />
                <Label htmlFor="r3">PDF</Label>
              </div>
            </RadioGroup>
          )}
        />
        <p className="text-xs text-foreground/70 mt-1">
          Select a format to export (CSV, Excel, or PDF).
        </p>
        <div className="flex justify-end gap-4 border-t pt-2 mt-5">
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
            disabled={watchedColumns?.length === 0}
          >
            Export
          </Button>
        </div>
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
          name={name}
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
          <Button variant="default" className="hidden h-8 lg:flex">
            <DotsVerticalIcon className="mr-2 h-4 w-4" /> Options
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-[150px]">
          <DropdownMenuLabel>Options</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => store.set("exportModalOpen", true)}>
            Import
          </DropdownMenuItem>
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
