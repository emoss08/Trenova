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

import { useMutation, useQuery, useQueryClient } from "react-query";
import {
  Box,
  Button,
  Card,
  Divider,
  Group,
  SimpleGrid,
  Skeleton,
  Text,
} from "@mantine/core";
import React from "react";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import {
  getEmailControl,
  getEmailProfiles,
} from "@/services/OrganizationRequestService";
import { usePageStyles } from "@/assets/styles/PageStyles";
import {
  EmailControl,
  EmailControlFormValues,
  EmailProfile,
} from "@/types/organization";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/lib/AxiosConfig";
import { APIError } from "@/types/server";
import { emailControlSchema } from "@/lib/schemas/OrganizationSchema";
import { SelectInput } from "@/components/common/fields/SelectInput";

interface Props {
  emailControl: EmailControl;
  selectEmailProfileData: TChoiceProps[];
}

function EmailControlForm({
  emailControl,
  selectEmailProfileData,
}: Props): React.ReactElement {
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
          data.errors.forEach((e: APIError) => {
            form.setFieldError(e.attr, e.detail);
            if (e.attr === "nonFieldErrors") {
              notifications.show({
                title: "Error",
                message: e.detail,
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
    },
  );

  const form = useForm<EmailControlFormValues>({
    validate: yupResolver(emailControlSchema),
    initialValues: {
      billingEmailProfile: emailControl.billingEmailProfile || "",
      rateExpirationEmailProfile: emailControl.rateExpirationEmailProfile || "",
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
            <SelectInput<EmailControlFormValues>
              form={form}
              data={selectEmailProfileData}
              className={classes.fields}
              name="billingEmailProfile"
              label="Billing Email Profile"
              placeholder="Billing Email Profile"
              description="The email profile that will be used for billing emails."
              variant="filled"
            />
            <SelectInput<EmailControlFormValues>
              form={form}
              data={selectEmailProfileData}
              className={classes.fields}
              name="rateExpirationEmailProfile"
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
}

export default function EmailControlPage() {
  const { classes } = usePageStyles();
  const queryClient = useQueryClient();

  const { data: emailProfileData, isLoading: isEmailProfilesLoading } =
    useQuery({
      queryKey: ["emailProfiles"],
      queryFn: () => getEmailProfiles(),
      initialData: () => queryClient.getQueryData(["emailProfiles"]),
      staleTime: Infinity,
    });

  const selectEmailProfileData =
    emailProfileData?.map((emailProfile: EmailProfile) => ({
      value: emailProfile.id,
      label: emailProfile.name,
    })) || [];

  const { data: emailControlData, isLoading: isEmailControlDataLoading } =
    useQuery({
      queryKey: ["emailControl"],
      queryFn: () => getEmailControl(),
      initialData: () => queryClient.getQueryData(["emailControl"]),
      staleTime: Infinity,
    });

  // Store first element of dispatchControlData in variable
  const emailControlDataArray = emailControlData?.[0];

  const isLoading = isEmailControlDataLoading || isEmailProfilesLoading;

  return isLoading ? (
    <Skeleton height={400} />
  ) : (
    <Card className={classes.card}>
      <Text fz="xl" fw={700} className={classes.text}>
        Email Controls
      </Text>

      <Divider my={10} />
      {emailControlDataArray && (
        <EmailControlForm
          emailControl={emailControlDataArray}
          selectEmailProfileData={selectEmailProfileData}
        />
      )}
    </Card>
  );
}
