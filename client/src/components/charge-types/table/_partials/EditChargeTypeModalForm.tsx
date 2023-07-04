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
import { Box, Button, Group } from "@mantine/core";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import { useMutation, useQueryClient } from "react-query";
import axios from "@/lib/AxiosConfig";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { APIError } from "@/types/server";
import { useForm, yupResolver } from "@mantine/form";
import { ChargeType, ChargeTypeFormValues } from "@/types/apps/billing";
import { chargeTypeTableStore } from "@/stores/BillingStores";
import { chargeTypeSchema } from "@/utils/apps/billing/schema";
import { useFormStyles } from "@/styles/FormStyles";

type Props = {
  chargeType: ChargeType;
};

export const EditChargeTypeModalForm: React.FC<Props> = ({ chargeType }) => {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: ChargeTypeFormValues) =>
      axios.put(`/charge_types/${chargeType.id}/`, values),
    {
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: ["charge-type-table-data"],
        });
        queryClient
          .invalidateQueries({
            queryKey: ["chargeType", chargeType.id],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Charge Type updated successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            chargeTypeTableStore.set("editModalOpen", false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((error: APIError) => {
            form.setFieldError(error.attr, error.detail);
            if (error.attr === "non_field_errors") {
              notifications.show({
                title: "Error",
                message: error.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    }
  );

  const form = useForm<ChargeTypeFormValues>({
    validate: yupResolver(chargeTypeSchema),
    initialValues: {
      name: chargeType.name,
      description: chargeType.description,
    },
  });

  const submitForm = (values: ChargeTypeFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <>
      <form onSubmit={form.onSubmit((values) => submitForm(values))}>
        <Box className={classes.div}>
          <Box>
            <ValidatedTextInput
              form={form}
              className={classes.fields}
              name="name"
              label="Name"
              placeholder="Name"
              variant="filled"
              withAsterisk
            />
            <ValidatedTextArea
              form={form}
              className={classes.fields}
              name="description"
              label="Description"
              placeholder="Description"
              variant="filled"
            />
            <Group position="right" mt="md">
              <Button
                color="white"
                type="submit"
                className={classes.control}
                loading={loading}
              >
                Submit
              </Button>
            </Group>
          </Box>
        </Box>
      </form>
    </>
  );
};
