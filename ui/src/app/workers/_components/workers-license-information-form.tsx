/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { AutoCompleteDateField } from "@/components/fields/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Separator } from "@/components/ui/separator";
import { endorsementChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { type WorkerSchema } from "@/lib/schemas/worker-schema";
import { Endorsement } from "@/types/worker";
import { useQuery } from "@tanstack/react-query";
import { useFormContext, useWatch } from "react-hook-form";

export default function WorkersLicenseInformationForm() {
  const { control } = useFormContext<WorkerSchema>();
  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data ?? [];

  // If the endorsement is H or T, then the hazmat expiry is required
  const hazmatExpiryRequired = useWatch({
    control,
    name: "profile.endorsement",
    defaultValue: Endorsement.None,
    exact: true,
  });

  const hazmatCheck =
    hazmatExpiryRequired === Endorsement.Hazmat ||
    hazmatExpiryRequired === Endorsement.TankerHazmat;

  return (
    <div className="size-full">
      <div className="flex select-none flex-col px-4">
        <h2 className="mt-2 text-2xl font-semibold">License Information</h2>
        <p className="text-xs text-muted-foreground">
          The following information is required for the worker to be able to
          work in the United States.
        </p>
      </div>
      <Separator className="mt-2" />
      <div className="p-4">
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="profile.licenseNumber"
              label="License Number"
              placeholder="License Number"
              description="The number of the worker's license"
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="profile.licenseStateId"
              label="License State"
              placeholder="License State"
              description="The state of the worker's license"
              options={usStateOptions}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              rules={{ required: true }}
              name="profile.licenseExpiry"
              label="License Expiry"
              description="The expiry date of the worker's license"
              placeholder="License Expiry"
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="profile.endorsement"
              label="Endorsement"
              placeholder="Endorsement"
              description="The endorsement of the worker's license"
              options={endorsementChoices}
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              control={control}
              clearable={!hazmatCheck}
              placeholder="Hazmat Expiry"
              rules={{
                required: hazmatCheck,
              }}
              name="profile.hazmatExpiry"
              label="Hazmat Expiry"
              description="The expiry date of the worker's hazmat endorsement"
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              placeholder="Last MVR Check"
              control={control}
              rules={{ required: true }}
              name="profile.lastMvrCheck"
              label="Last MVR Check"
              description="The last MVR check date of the worker"
            />
          </FormControl>
          <FormControl>
            <AutoCompleteDateField
              placeholder="Last Drug Test"
              control={control}
              rules={{ required: true }}
              name="profile.lastDrugTest"
              label="Last Drug Test"
              description="The last drug test date of the worker"
            />
          </FormControl>
        </FormGroup>
      </div>
    </div>
  );
}
