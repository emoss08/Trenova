import { type LocationFormValues as FormValues } from "@/types/location";
import { XIcon } from "lucide-react";
import { useFieldArray, type Control } from "react-hook-form";
import { InputField } from "../common/fields/input";
import { Button } from "../ui/button";
import { Form, FormControl, FormGroup } from "../ui/form";
import { ScrollArea } from "../ui/scroll-area";

export function LocationContactForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  const { fields, append, remove } = useFieldArray({
    control,
    name: "contacts",
    keyName: "id",
  });

  const handleAddContact = () => {
    append({ name: "", emailAddress: "", phoneNumber: "" });
  };

  return (
    <Form className="flex size-full flex-col">
      {fields.length > 0 ? (
        <>
          <ScrollArea className="h-[75vh] p-4">
            {fields.map((field, index) => (
              <FormGroup
                key={field.id}
                className="border-border rounded-md border border-dashed p-4 lg:grid-cols-3"
              >
                <FormControl>
                  <InputField
                    control={control}
                    name={`contacts.${index}.name`}
                    label="Name"
                    placeholder="Name"
                    description="Enter the full name of the primary contact for this location."
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl>
                  <InputField
                    control={control}
                    name={`contacts.${index}.emailAddress`}
                    label="Email Address"
                    placeholder="Email Address"
                    description="Provide the email address for direct communication with the location's contact."
                  />
                </FormControl>
                <FormControl>
                  <InputField
                    control={control}
                    name={`contacts.${index}.phoneNumber`}
                    label="Phone"
                    placeholder="Phone"
                    description="Input the telephone number for reaching the location's contact."
                  />
                </FormControl>
                <div className="flex max-w-sm flex-col justify-between gap-1">
                  <div className="min-h-[2em]">
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
            className="my-4 w-[200px]"
            onClick={handleAddContact}
          >
            Add Another Contact
          </Button>
        </>
      ) : (
        <div className="mt-44 flex grow flex-col items-center justify-center">
          <XIcon className="text-foreground size-10" />
          <h3 className="mt-4 text-lg font-semibold">
            No Location Contact added
          </h3>
          <p className="text-muted-foreground mb-4 mt-2 text-sm">
            You have not added any location contact. Add one below.
          </p>
          <Button type="button" size="sm" onClick={handleAddContact}>
            Add New Contact
          </Button>
        </div>
      )}
    </Form>
  );
}
