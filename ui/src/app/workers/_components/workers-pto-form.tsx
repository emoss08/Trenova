import { DoubleClickEditDate } from "@/components/fields/date-field";
import { DoubleClickSelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/ui/empty-state";
import { FormControl } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { Separator } from "@/components/ui/separator";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { ptoStatusChoices, ptoTypeChoices } from "@/lib/choices";
import { WorkerSchema } from "@/lib/schemas/worker-schema";
import { cn } from "@/lib/utils";
import { PTOStatus, PTOType } from "@/types/worker";
import {
  faCalendar,
  faClock,
  faTrash,
  faUser,
} from "@fortawesome/pro-regular-svg-icons";
import { useEffect, useState } from "react";
import {
  useFieldArray,
  UseFieldArrayRemove,
  useFormContext,
  useWatch,
} from "react-hook-form";

function WorkerPTOContent({
  index,
  remove,
}: {
  index: number;
  remove: UseFieldArrayRemove;
}) {
  const { control } = useFormContext<WorkerSchema>();
  const [showCancelForm, setShowCancelForm] = useState(false);
  const status = useWatch({
    control,
    name: `pto.${index}.status`,
  });
  const reason = useWatch({
    control,
    name: `pto.${index}.reason`,
  });

  // Watch for status changes
  useEffect(() => {
    if (status === PTOStatus.Cancelled && !reason) {
      setShowCancelForm(true);
    }
  }, [status, reason]);

  if (showCancelForm) {
    return (
      <PTOCancelForm
        index={index}
        onComplete={() => setShowCancelForm(false)}
      />
    );
  }

  return (
    <div className="relative grid size-full rounded-md border border-input p-2">
      <TooltipProvider>
        <Tooltip delayDuration={0}>
          <TooltipTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              className="absolute right-2 top-2 z-50"
              onClick={() => remove(index)}
            >
              <Icon icon={faTrash} className="size-4" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            <p>Remove</p>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>

      <div className="grid grid-cols-2 gap-1">
        <FormControl className="min-h-[3em]">
          <DoubleClickSelectField
            name={`pto.${index}.status`}
            control={control}
            rules={{ required: true }}
            options={ptoStatusChoices}
            placeholder="Status"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <DoubleClickSelectField
            name={`pto.${index}.type`}
            control={control}
            rules={{ required: true }}
            options={ptoTypeChoices}
            placeholder="Type"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <DoubleClickEditDate
            name={`pto.${index}.startDate`}
            control={control}
            rules={{ required: true }}
            placeholder="Start Date"
          />
        </FormControl>
        <FormControl className="min-h-[3em]">
          <DoubleClickEditDate
            name={`pto.${index}.endDate`}
            control={control}
            rules={{ required: true }}
            placeholder="End Date"
          />
        </FormControl>
      </div>
    </div>
  );
}

function PTOCancelForm({
  index,
  onComplete,
}: {
  index: number;
  onComplete: () => void;
}) {
  const { control, setValue } = useFormContext<WorkerSchema>();

  const handleCancel = () => {
    setValue(`pto.${index}.status`, PTOStatus.Requested);
    setValue(`pto.${index}.reason`, "");
    onComplete();
  };

  const handleConfirm = () => {
    setValue(`pto.${index}.status`, PTOStatus.Cancelled);
    onComplete();
  };

  return (
    <div className="grid gap-1 rounded-md border border-input p-2">
      <FormControl className="min-h-[3em]">
        <TextareaField
          control={control}
          name={`pto.${index}.reason`}
          label="Reason"
          placeholder="Reason for cancellation"
          rules={{ required: true }}
        />
      </FormControl>
      <div className="flex justify-end gap-2">
        <Button type="button" variant="outline" onClick={handleCancel}>
          Cancel
        </Button>
        <Button type="button" onClick={handleConfirm}>
          Confirm
        </Button>
      </div>
    </div>
  );
}

export default function WorkerPTOForm() {
  const { control } = useFormContext<WorkerSchema>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: "pto",
    keyName: "id",
  });

  const handleAddPTO = () => {
    append({
      endDate: 0,
      startDate: 0,
      reason: "",
      status: PTOStatus.Requested,
      type: PTOType.Vacation,
    });
  };

  return (
    <div className="size-full">
      <div className="flex select-none flex-col px-4">
        <h2 className="mt-2 text-2xl font-semibold">PTO Management</h2>
        <p className="text-xs text-muted-foreground">
          The following information is required for the worker to be able to
          work in the United States.
        </p>
      </div>
      <Separator className="mt-2" />
      <div className="h-[450px] w-full overflow-auto p-4">
        {fields.length > 0 ? (
          <>
            <div
              className={cn(
                "grid grid-cols-1 gap-2",
                fields.length > 1 && "grid-cols-2",
              )}
            >
              {fields.map((field, index) => (
                <WorkerPTOContent
                  key={field.id}
                  index={index}
                  remove={remove}
                />
              ))}
            </div>
            <div className="mt-4 grid grid-cols-2 gap-2 px-2">
              <div className="mx-2 my-4 flex justify-end">
                <Button type="button" onClick={handleAddPTO}>
                  Add PTO
                </Button>
              </div>
            </div>
          </>
        ) : (
          <div className="flex items-center justify-center">
            <EmptyState
              title="No PTO"
              description="Add a PTO to get started"
              className="size-full"
              icons={[faCalendar, faUser, faClock]}
              action={{
                label: "Add PTO",
                onClick: handleAddPTO,
              }}
            />
          </div>
        )}
      </div>
    </div>
  );
}
