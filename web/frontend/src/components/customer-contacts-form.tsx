/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { useFieldArray, useFormContext } from "react-hook-form";

import { InputField } from "@/components/common/fields/input";
import { type CustomerFormValues as FormValues } from "@/types/customer";
import { PersonIcon, PlusIcon } from "@radix-ui/react-icons";
import { useEffect, useState } from "react";
import { CheckboxInput } from "./common/fields/checkbox";
import { Button } from "./ui/button";
import { FormControl, FormGroup } from "./ui/form";
import { PhoneInput } from "./ui/phone-input";
import { ScrollArea } from "./ui/scroll-area";

export function CustomerContactForm() {
  const { control, watch } = useFormContext<FormValues>();
  const [emailRequired, setEmailRequired] = useState<boolean>(false);
  const { fields, append, remove } = useFieldArray({
    control,
    name: "contacts",
    keyName: "id",
  });

  const handleAddContact = () => {
    append({
      name: "",
      email: "",
      title: "",
      phoneNumber: "",
      isPayableContact: false,
    });
  };

  // Set Email field required when isPayableContact is true
  useEffect(() => {
    const subscription = watch((value, { name }) => {
      if (name?.startsWith("contacts") && name?.endsWith("isPayableContact")) {
        const contactIndex = Number(name.split(".")[1]);
        const isPayable =
          value.contacts?.[contactIndex]?.isPayableContact ?? false;
        setEmailRequired(isPayable);
      }
    });

    return () => subscription.unsubscribe();
  }, [watch, setEmailRequired]);

  return (
    <>
      <div className="flex size-full flex-col">
        {fields.length > 0 ? (
          <>
            <ScrollArea className="h-[75vh] p-4">
              {fields.map((field, index) => (
                <FormGroup
                  key={field.id}
                  className="border-border rounded-md border border-dashed p-4 lg:grid-cols-2"
                >
                  <FormControl>
                    <InputField
                      rules={{ required: true }}
                      control={control}
                      name={`contacts.${index}.name`}
                      description="Input the full name of the customer contact."
                      label="Name"
                      placeholder="Name"
                    />
                  </FormControl>
                  <FormControl>
                    <InputField
                      control={control}
                      name={`contacts.${index}.title`}
                      label="Title"
                      placeholder="Title"
                      description="Indicate the professional title of the customer contact."
                    />
                  </FormControl>
                  <FormControl>
                    <InputField
                      type="email"
                      rules={{ required: emailRequired }}
                      control={control}
                      name={`contacts.${index}.email`}
                      label="Email"
                      placeholder="Email"
                      description="Provide the customer contact's email address for correspondence."
                    />
                  </FormControl>
                  <FormControl>
                    <PhoneInput
                      control={control}
                      name={`contacts.${index}.phoneNumber`}
                      label="Phone"
                      placeholder="Phone"
                      description="Input the customer contact's telephone number for direct communication."
                    />
                  </FormControl>
                  <FormControl>
                    <CheckboxInput
                      control={control}
                      name={`contacts.${index}.isPayableContact`}
                      label="Is Payable Contact"
                      description="Check if the contact is responsible for managing payments and invoices."
                    />
                  </FormControl>
                  <div className="flex max-w-sm flex-col justify-between gap-1">
                    <div className="min-h-[4em]">
                      <Button
                        size="sm"
                        variant="linkHover2"
                        type="button"
                        onClick={() => remove(index)}
                      >
                        Remove
                      </Button>
                    </div>
                  </div>
                </FormGroup>
              ))}
            </ScrollArea>
            <Button
              type="button"
              size="sm"
              className="mb-10 w-fit"
              onClick={handleAddContact}
            >
              <PlusIcon className="mr-2 size-4" />
              Add Another Contacts
            </Button>
          </>
        ) : (
          <div className="mt-44 flex grow flex-col items-center justify-center">
            <PersonIcon className="text-foreground size-10" />
            <h3 className="mt-4 text-lg font-semibold">No Contacts added</h3>
            <p className="text-muted-foreground mb-4 mt-2 text-sm">
              You have not added any contacts. Add one below.
            </p>
            <Button type="button" size="sm" onClick={handleAddContact}>
              Add Contact
            </Button>
          </div>
        )}
      </div>
    </>
  );
}
