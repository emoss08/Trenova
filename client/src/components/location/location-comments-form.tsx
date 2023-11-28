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

import { useCommentTypes } from "@/hooks/useQueries";
import { AlertOctagonIcon } from "lucide-react";
import { SelectInput } from "../common/fields/select-input";
import { TextareaField } from "../common/fields/textarea";
import { Button } from "../ui/button";

export function LocationCommentForm({
  control,
}: {
  control: Control<FormValues>;
}) {
  const { fields, append, remove } = useFieldArray({
    control,
    name: "locationComments",
    keyName: "id",
  });

  const handleAddContact = () => {
    append({ commentType: "", comment: "" });
  };

  const {
    selectCommentTypes,
    isError: isCommentTypeError,
    isLoading: isCommentTypeLoading,
  } = useCommentTypes();

  return (
    <div className="flex flex-col h-full w-full">
      {fields.length > 0 ? (
        <>
          <div className="max-h-[600px] overflow-y-auto">
            {fields.map((field, index) => (
              <div
                key={field.id}
                className="grid grid-cols-1 gap-2 my-4 pb-2 border-b"
              >
                <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
                  <div className="min-h-[4em]">
                    <SelectInput
                      rules={{ required: true }}
                      name={`locationComments.${index}.commentType`}
                      control={control}
                      label="Comment Type"
                      options={selectCommentTypes}
                      isLoading={isCommentTypeLoading}
                      isFetchError={isCommentTypeError}
                      placeholder="Comment Type"
                      description="Select the type of comment."
                    />
                  </div>
                </div>
                <div className="flex flex-col justify-between w-full max-w-sm gap-0.5">
                  <div className="min-h-[4em]">
                    <TextareaField
                      rules={{ required: true }}
                      name={`locationComments.${index}.comment`}
                      control={control}
                      label="Comment"
                      placeholder="Comment"
                      description="Additional notes or comments for the account"
                    />
                  </div>
                </div>
                <div className="flex flex-col justify-between max-w-sm gap-1">
                  <div className="min-h-[2em]">
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
            className="mb-10 w-[200px]"
            onClick={handleAddContact}
          >
            Add Another Comment
          </Button>
        </>
      ) : (
        <div className="flex-grow flex flex-col items-center justify-center mt-48">
          <span className="text-6xl mb-4">
            <AlertOctagonIcon />
          </span>
          <p className="mb-4">No comments yet. Please add a new comment.</p>
          <Button type="button" size="sm" onClick={handleAddContact}>
            Add Comment
          </Button>
        </div>
      )}
    </div>
  );
}
