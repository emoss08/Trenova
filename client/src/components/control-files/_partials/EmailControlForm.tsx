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

import { Box, Button, Group, SimpleGrid } from "@mantine/core";
import React from "react";
import { useForm, yupResolver } from "@mantine/form";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import axios from "@/lib/AxiosConfig";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { APIError } from "@/types/server";
import {
  EmailControl,
  EmailControlFormValues,
} from "@/types/apps/organization";
import { TChoiceProps } from "@/types";
import { emailControlSchema } from "@/utils/apps/organization/schema";
import { useFormStyles } from "@/styles/FormStyles";

interface Props {
  emailControl: EmailControl;
  selectEmailProfileData: TChoiceProps[];
}

export const EmailControlForm: React.FC<Props> = ({
  emailControl,
  selectEmailProfileData,
}) => {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: EmailControlFormValues) =>
      axios.put(`/email_control/${emailControl.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["emailControl"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Email Control updated successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
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

  const form = useForm<EmailControlFormValues>({
    validate: yupResolver(emailControlSchema),
    initialValues: {
      billing_email_profile: emailControl.billing_email_profile || "",
      rate_expiration_email_profile:
        emailControl.rate_expiration_email_profile || "",
    },
  });

  const handleSubmit = (values: EmailControlFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={form.onSubmit((values) => handleSubmit(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput
              form={form}
              data={selectEmailProfileData}
              className={classes.fields}
              name="billing_email_profile"
              label="Billing Email Profile"
              placeholder="Billing Email Profile"
              description="The email profile that will be used for billing emails."
              variant="filled"
            />
            <SelectInput
              form={form}
              data={selectEmailProfileData}
              className={classes.fields}
              name="rate_expiration_email_profile"
              label="Rate Expiration Email Profile"
              placeholder="Rate Expiration Email Profile"
              description="The email profile that will be used for rate expiration emails."
              variant="filled"
            />
          </SimpleGrid>
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
  );
};
