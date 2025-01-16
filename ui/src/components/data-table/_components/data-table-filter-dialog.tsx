import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
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
import { dataTableConfig } from "@/config/data-table";
import { generateRowId } from "@/hooks/use-data-table-query";
import { getDefaultFilterOperator, getValidFilters } from "@/lib/data-table";
import { http } from "@/lib/http-client";
import { Status } from "@/types/common";
import {
  DataTableAdvancedFilterField,
  Filter,
  StringKeyOf,
} from "@/types/data-table";
import { APIError } from "@/types/errors";
import { Visibility } from "@/types/table-configuration";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import {
  type ColumnFiltersState,
  SortingState,
  Table,
} from "@tanstack/react-table";
import { FormProvider, useForm, useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

interface DataTableFilterDialogProps<TData> {
  open: boolean;
  onClose: () => void;
  table: Table<TData>;
  columnFilters: ColumnFiltersState;
  sorting: SortingState;
  filterFields: DataTableAdvancedFilterField<TData>[];
}

const filterConfigSchema = z.object({
  filters: z.array(
    z.object({
      id: z.string(),
      value: z.union([z.string(), z.array(z.string())]),
      type: z.enum(dataTableConfig.columnTypes),
      operator: z.enum(dataTableConfig.globalOperators),
      rowId: z.string(),
    }),
  ),
  joinOperator: z.enum(["and", "or"]).default("and"),
  pageSize: z.number(),
});

function transformColumnFilter<TData>(
  filter: ColumnFiltersState[number],
  filterFields: DataTableAdvancedFilterField<TData>[],
): Filter<TData> {
  const fieldConfig = filterFields.find((f) => f.id === filter.id);

  let filterValue: string | string[];
  if (Array.isArray(filter.value)) {
    filterValue = filter.value.map(String);
  } else {
    filterValue = String(filter.value ?? "");
  }

  return {
    id: filter.id as StringKeyOf<TData>,
    value: filterValue,
    operator:
      (filter as any).operator ||
      getDefaultFilterOperator(fieldConfig?.type || "text"),
    type: fieldConfig?.type || "text",
    rowId: (filter as any).rowId || generateRowId(),
  };
}

const filterSchema = z.object({
  status: z.nativeEnum(Status).default(Status.Active),
  name: z.string().min(1, "Name is required"),
  description: z.string().optional(),
  tableIdentifier: z.string().min(1, "Table identifier is required"),
  filterConfig: filterConfigSchema,
  visibility: z.nativeEnum(Visibility).default(Visibility.Private),
});

type FilterFormValues = z.infer<typeof filterSchema>;

function DataTableFilterForm<TData>({
  onClose,
  table,
  columnFilters,
  sorting,
  filterFields,
}: DataTableFilterDialogProps<TData>) {
  const {
    control,
    handleSubmit,
    setError,
    reset,
    getValues,
    formState: { isSubmitting, errors },
  } = useFormContext<FilterFormValues>();
  console.info("values", getValues());
  console.info("errors", errors);

  const mutation = useMutation({
    mutationFn: async (values: FilterFormValues) => {
      const response = await http.post("/table-configurations", values);
      return response.data;
    },
    onSuccess: () => {
      toast.success("Filter created");
      onClose();
      reset();
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as keyof FilterFormValues, {
            message: fieldError.reason,
          });
        });
      }
      toast.error(error.message || "Failed to create filter");
    },
  });

  async function onSubmit(values: FilterFormValues) {
    const transformedFilters = columnFilters.map((filter) =>
      transformColumnFilter<TData>(filter, filterFields),
    );

    const validFilters = getValidFilters(transformedFilters);

    const configWithFilters: FilterFormValues = {
      ...values,
      filterConfig: {
        filters: validFilters,
        joinOperator: "and",
        sorting: sorting.map((sort) => ({
          id: sort.id,
          desc: sort.desc,
        })),
        pageSize: table.getState().pagination.pageSize,
      },
    };

    console.info("values", JSON.stringify(configWithFilters));

    await mutation.mutateAsync(configWithFilters);
  }

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <FormGroup cols={1}>
        <FormControl>
          <InputField
            rules={{ required: true }}
            control={control}
            name="name"
            label="Name"
            placeholder="Name"
            description="Unique name for this filter"
          />
        </FormControl>
        <FormControl>
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Description"
            description="Description for this filter"
          />
        </FormControl>
      </FormGroup>
      <DialogFooter>
        <Button type="submit" isLoading={isSubmitting}>
          Save
        </Button>
      </DialogFooter>
    </Form>
  );
}

export function DataTableFilterDialog<TData>({
  open,
  onClose,
  table,
  columnFilters,
  sorting,
  filterFields,
}: DataTableFilterDialogProps<TData>) {
  const form = useForm<FilterFormValues>({
    resolver: zodResolver(filterSchema),
    defaultValues: {
      status: Status.Active,
      name: "",
      description: "",
      filterConfig: {
        filters: [],
        joinOperator: "and",
        pageSize: 20,
      },
      tableIdentifier: window.location.pathname,
      visibility: Visibility.Private,
    },
  });

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Filter</DialogTitle>
          <DialogDescription>
            Save your current filters and sorting preferences for future use.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <DataTableFilterForm
            open={open}
            onClose={onClose}
            table={table}
            columnFilters={columnFilters}
            sorting={sorting}
            filterFields={filterFields}
          />
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
