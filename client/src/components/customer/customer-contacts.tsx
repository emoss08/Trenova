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

import { Control, useFieldArray } from "react-hook-form";
import { InputField } from "../common/fields/input";

import { yesAndNoChoicesBoolean } from "@/lib/constants";
import { CustomerFormValues as FormValues } from "@/types/customer";
import { PlusIcon } from "@radix-ui/react-icons";
import { AlertOctagonIcon } from "lucide-react";
import { CheckboxInput } from "../common/fields/checkbox";
import { SelectInput } from "../common/fields/select-input";
import { Button } from "../ui/button";

export function CustomerContactForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  const { fields, append, remove } = useFieldArray({
    control,
    name: "customerContacts",
    keyName: "id",
  });

  const handleAddContact = () => {
    append({
      isActive: true,
      name: "",
      email: "",
      title: "",
      phone: "",
      isPayableContact: false,
    });
  };

  return (
    <>
      <div className="flex flex-col h-full w-full">
        {fields.length > 0 ? (
          <>
            <div className="max-h-[600px] overflow-y-auto">
              {fields.map((field, index) => (
                <div
                  key={field.id}
                  className="grid grid-cols-2 gap-2 my-4 pb-2 border-b"
                >
                  <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
                    <div className="min-h-[4em]">
                      <SelectInput
                        name={`customerContacts.${index}.isActive`}
                        rules={{ required: true }}
                        control={control}
                        label="Status"
                        options={yesAndNoChoicesBoolean}
                        description="Select the current status of the customer contact's activity."
                        placeholder="Select Status"
                        isClearable={false}
                        menuPlacement="bottom"
                        menuPosition="fixed"
                      />
                    </div>
                  </div>
                  <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
                    <div className="min-h-[4em]">
                      <InputField
                        rules={{ required: true }}
                        control={control}
                        name={`customerContacts.${index}.name`}
                        description="Input the full name of the customer contact."
                        label="Name"
                        placeholder="Name"
                      />
                    </div>
                  </div>
                  <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
                    <div className="min-h-[4em]">
                      <InputField
                        type="email"
                        rules={{ required: true }}
                        control={control}
                        name={`customerContacts.${index}.email`}
                        label="Email"
                        placeholder="Email"
                        description="Provide the customer contact's email address for correspondence."
                      />
                    </div>
                  </div>
                  <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
                    <div className="min-h-[4em]">
                      <InputField
                        rules={{ required: true }}
                        control={control}
                        name={`customerContacts.${index}.title`}
                        label="Title"
                        placeholder="Title"
                        description="Indicate the professional title of the customer contact."
                      />
                    </div>
                  </div>
                  <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
                    <div className="min-h-[4em]">
                      <InputField
                        control={control}
                        name={`customerContacts.${index}.phone`}
                        label="Phone"
                        placeholder="Phone"
                        description="Input the customer contact's telephone number for direct communication."
                      />
                    </div>
                  </div>
                  <div className="flex flex-col justify-between w-full max-w-sm mt-6 gap-0.5">
                    <div className="min-h-[4em]">
                      <CheckboxInput
                        control={control}
                        name={`customerContacts.${index}.isPayableContact`}
                        label="Is Payable Contact"
                        description="Check if the contact is responsible for managing payments and invoices."
                      />
                    </div>
                  </div>
                  <div className="flex flex-col justify-between max-w-sm gap-1">
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
              className="mb-10 w-fit"
              onClick={handleAddContact}
            >
              <PlusIcon className="mr-2 h-4 w-4" />
              Add Another Contacts
            </Button>
          </>
        ) : (
          <div className="flex-grow flex flex-col items-center justify-center mt-44">
            <span className="text-6xl mb-4">
              <AlertOctagonIcon />
            </span>
            <p className="mb-4">No contacts yet. Please add a new contacts.</p>
            <Button type="button" size="sm" onClick={handleAddContact}>
              Add Contact
            </Button>
          </div>
        )}
      </div>
    </>
  );
}
