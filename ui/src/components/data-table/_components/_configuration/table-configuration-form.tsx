import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import {
  Sortable,
  SortableDragHandle,
  SortableItem,
} from "@/components/ui/sortable";
import { Switch } from "@/components/ui/switch";
import { visibilityChoices } from "@/lib/choices";
import {
  getFilterOperatorLabel,
  getSortDirectionLabel,
} from "@/lib/data-table-utils";
import { TableConfigurationSchema } from "@/lib/schemas/table-configuration-schema";
import { GripVertical } from "lucide-react";
import { Controller, useFormContext, useWatch } from "react-hook-form";

function formatColumnName(name: string) {
  const result = name.replace(/([A-Z])/g, " $1");
  return result.charAt(0).toUpperCase() + result.slice(1);
}

export function TableConfigurationForm() {
  const { control, register, setValue } =
    useFormContext<TableConfigurationSchema>();

  const columnVisibilityKeys = useWatch({
    control,
    name: "tableConfig.columnVisibility",
  });

  const columnOrder = useWatch({
    control,
    name: "tableConfig.columnOrder",
  });

  const filters = useWatch({
    control,
    name: "tableConfig.filters",
  });

  const sort = useWatch({
    control,
    name: "tableConfig.sort",
  });

  return (
    <div className="flex flex-col gap-3">
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            options={visibilityChoices}
            rules={{ required: true }}
            name="visibility"
            label="Visibility"
            description="The visibility of the table configuration."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="name"
            label="Name"
            placeholder="Name"
            description="The name of the table configuration."
          />
        </FormControl>
        <FormControl cols="full">
          <TextareaField
            control={control}
            name="description"
            label="Description"
            placeholder="Description"
            description="The description of the table configuration."
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            outlined
            name="isDefault"
            label="Default"
            description="When enabled, the system will automatically apply this table configuration to the table when the user first navigates to it."
            position="left"
          />
        </FormControl>
        <input type="hidden" {...register("tableConfig")} />
      </FormGroup>
      <div className="flex flex-col gap-3 bg-muted rounded-md p-2 border border-border border-dashed">
        <h3 className="text-sm text-center font-medium border-b border-border border-dashed pb-2">
          Column Visibility
        </h3>
        {columnVisibilityKeys &&
        Object.keys(columnVisibilityKeys).length > 0 ? (
          Object.keys(columnVisibilityKeys).map((key) => (
            <Controller
              key={key}
              name={`tableConfig.columnVisibility.${key}`}
              control={control}
              render={({ field }) => (
                <div className="flex items-center justify-between">
                  <Label
                    htmlFor={`col-vis-${key}`}
                    className="flex-grow text-xs"
                  >
                    {formatColumnName(key)}
                  </Label>
                  <Switch
                    id={`col-vis-${key}`}
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    size="sm"
                  />
                </div>
              )}
            />
          ))
        ) : (
          <p className="text-sm text-center text-muted-foreground py-2">
            No column visibility options to configure for this table.
          </p>
        )}
      </div>
      <div className="flex flex-col gap-3 bg-muted rounded-md p-2 border border-border border-dashed">
        <h3 className="text-sm text-center font-medium border-b border-border border-dashed pb-2">
          Filters
        </h3>
        {filters && filters.length > 0 ? (
          <div className="flex flex-col">
            {filters.map((filter, index) => (
              <div key={filter.field}>
                <div className="grid grid-cols-3 gap-4 items-start">
                  <FilterRow
                    label="Field"
                    value={formatColumnName(filter.field)}
                  />
                  <FilterRow
                    label="Operator"
                    value={getFilterOperatorLabel(filter.operator)}
                  />
                  <FilterRow label="Value" value={filter.value} />
                </div>
                {index < filters.length - 1 && (
                  <div className="flex justify-center py-2">
                    <span className="px-3 py-1 bg-background border border-border rounded-md text-xs font-medium text-muted-foreground">
                      AND
                    </span>
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-center text-muted-foreground py-2">
            No filters to configure for this table.
          </p>
        )}
      </div>
      <div className="flex flex-col gap-3 bg-muted rounded-md p-2 border border-border border-dashed">
        <h3 className="text-sm text-center font-medium border-b border-border border-dashed pb-2">
          Sort
        </h3>
        {sort && sort.length > 0 ? (
          <div className="flex flex-col">
            {sort.map((filter, index) => (
              <div key={filter.field}>
                <div className="grid grid-cols-2 gap-4 items-center">
                  <FilterRow
                    label="Field"
                    value={formatColumnName(filter.field)}
                  />
                  <FilterRow
                    label="Direction"
                    value={getSortDirectionLabel(filter.direction)}
                  />
                </div>
                {index < sort.length - 1 && (
                  <div className="flex justify-center py-2">
                    <span className="px-3 py-1 bg-background border border-border rounded-md text-xs font-medium text-muted-foreground">
                      AND
                    </span>
                  </div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-center text-muted-foreground py-2">
            No sort to configure for this table.
          </p>
        )}
      </div>
      <div className="flex flex-col gap-3 bg-muted rounded-md p-2 border border-border border-dashed">
        <h3 className="text-sm text-center font-medium border-b border-border border-dashed pb-2">
          Column Order
        </h3>
        {columnOrder && columnOrder.length > 0 ? (
          <Sortable
            value={columnOrder.map((id) => ({ id }))}
            onValueChange={(newOrder) =>
              setValue(
                "tableConfig.columnOrder",
                newOrder.map((item) => item.id as string),
              )
            }
            orientation="vertical"
          >
            <div className="flex flex-col gap-2">
              {columnOrder.map((columnId) => (
                <SortableItem key={columnId} value={columnId}>
                  <div className="flex items-center gap-2 px-2 py-1.5 bg-background border border-border rounded hover:bg-accent transition-colors">
                    <SortableDragHandle
                      variant="ghost"
                      size="icon"
                      className="h-6 w-6 shrink-0"
                      type="button"
                      onClick={(e) => {
                        e.preventDefault();
                        e.stopPropagation();
                      }}
                    >
                      <GripVertical className="h-3 w-3" />
                    </SortableDragHandle>
                    <span className="text-sm flex-1">
                      {formatColumnName(columnId)}
                    </span>
                  </div>
                </SortableItem>
              ))}
            </div>
          </Sortable>
        ) : (
          <p className="text-sm text-center text-muted-foreground py-2">
            No column order configured for this table.
          </p>
        )}
      </div>
    </div>
  );
}

function FilterRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-1 text-center">
      <h3 className="text-xs font-medium uppercase text-muted-foreground">
        {label}
      </h3>
      <div className="px-2 py-1 bg-background border border-border rounded text-sm">
        {value}
      </div>
    </div>
  );
}
