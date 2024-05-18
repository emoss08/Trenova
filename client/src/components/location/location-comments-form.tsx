import { type LocationFormValues as FormValues } from "@/types/location";
import { useFieldArray, type Control } from "react-hook-form";

import { useCommentTypes } from "@/hooks/useQueries";
import { useUserStore } from "@/stores/AuthStore";
import { XIcon } from "lucide-react";
import { SelectInput } from "../common/fields/select-input";
import { TextareaField } from "../common/fields/textarea";
import { Button } from "../ui/button";
import { Form, FormControl, FormGroup } from "../ui/form";
import { ScrollArea } from "../ui/scroll-area";

export function LocationCommentForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  const user = useUserStore.get("user");
  const { fields, append, remove } = useFieldArray({
    control,
    name: "comments",
    keyName: "id",
  });

  const handleAddContact = () => {
    append({ commentTypeId: "", comment: "", userId: user.id });
  };

  const {
    selectCommentTypes,
    isError: isCommentTypeError,
    isLoading: isCommentTypeLoading,
  } = useCommentTypes();

  return (
    <Form className="flex size-full flex-col">
      {fields.length > 0 ? (
        <>
          <ScrollArea className="h-[70vh] p-4">
            {fields.map((field, index) => (
              <FormGroup
                key={field.id}
                className="border-border mb-4 grid grid-cols-2 gap-2 rounded-md border border-dashed p-4 lg:grid-cols-2"
              >
                <FormControl className="col-span-full">
                  <SelectInput
                    rules={{ required: true }}
                    name={`comments.${index}.commentTypeId`}
                    control={control}
                    label="Comment Type"
                    options={selectCommentTypes}
                    isLoading={isCommentTypeLoading}
                    isFetchError={isCommentTypeError}
                    placeholder="Comment Type"
                    description="Specify the category of the comment from the available options."
                    popoutLink="/dispatch/comment-types/"
                    hasPopoutWindow
                    popoutLinkLabel="Comment Type"
                  />
                </FormControl>
                <FormControl className="col-span-full">
                  <TextareaField
                    rules={{ required: true }}
                    name={`comments.${index}.comment`}
                    control={control}
                    label="Comment"
                    placeholder="Comment"
                    description="Provide detailed remarks or observations relevant to the account."
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
            Add Another Comment
          </Button>
        </>
      ) : (
        <div className="mt-44 flex grow flex-col items-center justify-center">
          <XIcon className="text-foreground size-10" />
          <h3 className="mt-4 text-lg font-semibold">
            No Location Comment added
          </h3>
          <p className="text-muted-foreground mb-4 mt-2 text-sm">
            You have not added any location comment. Add one below.
          </p>
          <Button type="button" size="sm" onClick={handleAddContact}>
            Add New Comment
          </Button>
        </div>
      )}
    </Form>
  );
}
