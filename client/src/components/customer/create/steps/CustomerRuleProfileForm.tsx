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

import React from "react";
import { Box } from "@mantine/core";
import { UseFormReturnType } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useQueryClient } from "@tanstack/react-query";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { CreateCustomerFormValues } from "@/types/customer";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedMultiSelect } from "@/components/common/fields/MultiSelect";
import axios from "@/lib/axiosConfig";
import { APIError } from "@/types/server";
import { TChoiceProps } from "@/types";

export function CustomerRuleProfileForm({
  documentClasses,
  isDocumentClassesLoading,
  isDocumentClassesError,
  form,
}: {
  documentClasses: Array<TChoiceProps>;
  isDocumentClassesLoading: boolean;
  isDocumentClassesError: boolean;
  form: UseFormReturnType<CreateCustomerFormValues>;
}): React.ReactElement {
  const { classes } = useFormStyles();
  const queryClient = useQueryClient();

  return (
    <Box className={classes.div}>
      <ValidatedTextInput<CreateCustomerFormValues>
        form={form}
        name="ruleProfile.name"
        label="Name"
        placeholder="Enter name"
        variant="filled"
        withAsterisk
      />
      <ValidatedMultiSelect<CreateCustomerFormValues>
        form={form}
        name="ruleProfile.documentClass"
        data={documentClasses}
        placeholder="Select document class"
        label="Document Class"
        isLoading={isDocumentClassesLoading}
        isError={isDocumentClassesError}
        variant="filled"
        withAsterisk
        creatable
        getCreateLabel={(query) => `+ Create ${query}`}
        onCreate={(query) => {
          // This is a reference to the object that will be updated asynchronously.
          const item = {
            value: "", // or some default value
            label: "", // or some default value
          };

          axios
            .post("/document_classifications/", { name: query })
            .then(async (response) => {
              if (response.status === 201) {
                await queryClient.invalidateQueries({
                  queryKey: ["documentClassifications"],
                });

                notifications.show({
                  title: "Success",
                  message: "Document Classification created successfully",
                  color: "green",
                  withCloseButton: true,
                  icon: <FontAwesomeIcon icon={faCheck} />,
                });

                // Update the properties of the item reference
                item.value = response.data.id;
                item.label = response.data.name;
              }
            })
            .catch((error) => {
              const { data } = error.response;
              if (data.type === "validation_error") {
                data.errors.forEach((e: APIError) => {
                  notifications.show({
                    title: "Error",
                    message: e.detail,
                    color: "red",
                    withCloseButton: true,
                    icon: <FontAwesomeIcon icon={faXmark} />,
                    autoClose: 10_000, // 10 seconds
                  });
                });
              }
            });
          return item;
        }}
      />
    </Box>
  );
}
