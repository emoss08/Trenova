import { useFieldArray, useFormContext } from "react-hook-form";

import { InputField } from "@/components/common/fields/input";
import { SelectInput } from "@/components/common/fields/select-input";
import { statusChoices } from "@/lib/choices";
import { type CustomerFormValues as FormValues } from "@/types/customer";
import { PersonIcon, PlusIcon } from "@radix-ui/react-icons";
import { useEffect, useState } from "react";
import { CheckboxInput } from "./common/fields/checkbox";
import { Button } from "./ui/button";
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
      status: "A",
      name: "",
      email: "",
      title: "",
      phone: "",
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
            <ScrollArea className="h-[65vh] p-4">
              {fields.map((field, index) => (
                <div
                  key={field.id}
                  className="border-border my-4 grid grid-cols-2 gap-2 rounded-md border p-2"
                >
                  <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <SelectInput
                        name={`contacts.${index}.status`}
                        rules={{ required: true }}
                        control={control}
                        label="Status"
                        options={statusChoices}
                        description="Select the current status of the customer contact's activity."
                        placeholder="Select Status"
                        isClearable={false}
                        menuPlacement="bottom"
                        menuPosition="fixed"
                      />
                    </div>
                  </div>
                  <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <InputField
                        rules={{ required: true }}
                        control={control}
                        name={`contacts.${index}.name`}
                        description="Input the full name of the customer contact."
                        label="Name"
                        placeholder="Name"
                      />
                    </div>
                  </div>
                  <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <InputField
                        control={control}
                        name={`contacts.${index}.title`}
                        label="Title"
                        placeholder="Title"
                        description="Indicate the professional title of the customer contact."
                      />
                    </div>
                  </div>
                  <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <InputField
                        type="email"
                        rules={{ required: emailRequired }}
                        control={control}
                        name={`contacts.${index}.email`}
                        label="Email"
                        placeholder="Email"
                        description="Provide the customer contact's email address for correspondence."
                      />
                    </div>
                  </div>
                  <div className="flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <InputField
                        control={control}
                        name={`contacts.${index}.phone`}
                        label="Phone"
                        placeholder="Phone"
                        description="Input the customer contact's telephone number for direct communication."
                      />
                    </div>
                  </div>
                  <div className="mt-6 flex w-full max-w-sm flex-col justify-between gap-0.5">
                    <div className="min-h-[4em]">
                      <CheckboxInput
                        control={control}
                        name={`contacts.${index}.isPayableContact`}
                        label="Is Payable Contact"
                        description="Check if the contact is responsible for managing payments and invoices."
                      />
                    </div>
                  </div>
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
                </div>
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
