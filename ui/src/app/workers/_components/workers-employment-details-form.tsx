import { AsyncSelectField } from "@/components/fields/async-select";
import { AutoCompleteDateField } from "@/components/fields/date-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { genderChoices, statusChoices, workerTypeChoices } from "@/lib/choices";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { useFormContext } from "react-hook-form";

export default function WorkersEmploymentDetailsForm() {
  const { control } = useFormContext<WorkerSchema>();

  return (
    <div className="size-full">
      <div className="flex select-none flex-col px-4">
        <h2 className="mt-2 text-2xl font-semibold">Employment Details</h2>
        <p className="text-xs text-muted-foreground">
          The following information is required for the worker to be able to
          work in the United States.
        </p>
      </div>
      <Separator className="mt-2" />
      <div className="p-4">
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label="Status"
              placeholder="Status"
              description="The status of the worker"
              options={statusChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="type"
              label="Type"
              placeholder="Type"
              description="The type of the worker"
              options={workerTypeChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="gender"
              label="Gender"
              placeholder="Gender"
              description="The gender of the worker"
              options={genderChoices}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              rules={{ required: true }}
              name="profile.dob"
              label="Date of Birth"
              description="The date of birth of the worker"
              placeholder="Date of Birth"
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              rules={{ required: true }}
              name="profile.hireDate"
              label="Hire Date"
              description="The date of hire of the worker"
              placeholder="Hire Date"
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              name="profile.terminationDate"
              label="Termination Date"
              description="The date of termination of the worker"
              placeholder="Termination Date"
            />
          </FormControl>
          <FormControl>
            <AsyncSelectField
              name="fleetCodeId"
              control={control}
              link="/fleet-codes"
              label="Fleet Code"
              placeholder="Select Fleet Code"
              description="Select the fleet code of the worker"
              hasPopoutWindow
              popoutLink="/fleet-codes/"
              popoutLinkLabel="Fleet Code"
            />
          </FormControl>
        </FormGroup>
      </div>
    </div>
  );
}
