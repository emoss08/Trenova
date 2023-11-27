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

import { LocationFormValues as FormValues } from "@/types/location";
import { Control, useFieldArray } from "react-hook-form";
import { InputField } from "../common/fields/input";
import { Button } from "../ui/button";

export function LocationContactForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  const { fields, append, remove } = useFieldArray({
    control,
    name: "locationContacts",
    keyName: "id",
  });

  return (
    <div>
      <div className="max-h-[600px] overflow-y-auto">
        {fields.map((field, index) => (
          <div
            key={field.id}
            className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2 my-4 pb-2 border-b"
          >
            <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
              <div className="min-h-[4em]">
                <InputField
                  control={control}
                  name={`locationContacts.${index}.name`}
                  label="Name"
                  placeholder="Name"
                  description="Enter the full name of the primary contact for this location."
                  rules={{ required: true }}
                />
              </div>
            </div>
            <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
              <div className="min-h-[4em]">
                <InputField
                  control={control}
                  name={`locationContacts.${index}.email`}
                  label="Email"
                  placeholder="Email"
                  description="Provide the email address for direct communication with the location's contact."
                />
              </div>
            </div>
            <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
              <div className="min-h-[4em]">
                <InputField
                  control={control}
                  name={`locationContacts.${index}.phone`}
                  label="Phone"
                  placeholder="Phone"
                  description="Input the telephone number for reaching the location's contact."
                />
              </div>
            </div>
            <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
              <div className="min-h-[4em]">
                <InputField
                  control={control}
                  name={`locationContacts.${index}.fax`}
                  label="Fax"
                  placeholder="Fax"
                  description="If applicable, list the fax number associated with the location's contact."
                />
              </div>
            </div>
            <div className="flex flex-col justify-between w-full max-w-sm mt-6 gap-1">
              <div className="min-h-[4em]">
                <Button
                  size="sm"
                  className="bg-background text-red-600 hover:bg-background hover:text-red-700"
                  type="button"
                  onClick={() => remove(index)}
                >
                  Remove
                </Button>
              </div>
            </div>
          </div>
        ))}
      </div>
      <Button
        type="button"
        size="sm"
        className="mb-10"
        onClick={() =>
          append({ name: "", email: "", phone: "", fax: undefined })
        }
      >
        Add Another Contact
      </Button>
    </div>
  );
}
