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
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, yupResolver } from "@mantine/form";
import {
  Box,
  Button,
  Card,
  Divider,
  Group,
  SimpleGrid,
  Text,
} from "@mantine/core";
import { useFormStyles } from "@/styles/FormStyles";
import {
  CustomerEmailProfile,
  CustomerEmailProfileFormValues,
} from "@/types/apps/customer";
import axios from "@/lib/AxiosConfig";
import { APIError } from "@/types/server";
import { CustomerEmailProfileSchema } from "@/utils/apps/customers/schema";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { SwitchInput } from "@/components/ui/fields/SwitchInput";
import { usePageStyles } from "@/styles/PageStyles";

type Props = {
  emailProfile: CustomerEmailProfile;
};

export function CustomerEmailProfileForm({
  emailProfile,
}: Props): React.ReactElement {
  const { classes } = useFormStyles();
  const { classes: pageClass } = usePageStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: CustomerEmailProfileFormValues) =>
      axios.patch(`/customer_email_profiles/${emailProfile?.id}/`, values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["customerEmailProfile", emailProfile?.id],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Customer Email Profile updated successfully",
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
            if (e.attr === "non_field_errors") {
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

  const form = useForm<CustomerEmailProfileFormValues>({
    validate: yupResolver(CustomerEmailProfileSchema),
    initialValues: {
      subject: emailProfile?.subject || "",
      comment: emailProfile?.comment || "",
      fromAddress: emailProfile?.fromAddress || "",
      blindCopy: emailProfile?.blindCopy || "",
      readReceipt: emailProfile?.readReceipt || false,
      readReceiptTo: emailProfile?.readReceiptTo || "",
      attachmentName: emailProfile?.attachmentName || "",
    },
  });

  const submitForm = (values: CustomerEmailProfileFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  return (
    <Card className={pageClass.card}>
      <Box>
        <Text className={classes.text} fw={600} fz={20}>
          Customer Email Profile
        </Text>
        <form onSubmit={form.onSubmit((values) => submitForm(values))}>
          <Box className={classes.div}>
            <Divider my={10} />
            <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
              <Box>
                <ValidatedTextInput<CustomerEmailProfileFormValues>
                  label="Subject"
                  placeholder="Subject"
                  form={form}
                  name="subject"
                  maxLength={100}
                  variant="filled"
                />
                <Text size="xs" color="dimmed">
                  Subject for the email
                </Text>
              </Box>
              <Box>
                <ValidatedTextInput<CustomerEmailProfileFormValues>
                  label="Comment"
                  placeholder="Comment"
                  form={form}
                  name="comment"
                  maxLength={100}
                  variant="filled"
                />
                <Text size="xs" color="dimmed">
                  Comment for the email
                </Text>
              </Box>
              <Box>
                <ValidatedTextInput<CustomerEmailProfileFormValues>
                  label="From Address"
                  placeholder="From Address"
                  form={form}
                  name="fromAddress"
                  variant="filled"
                />
                <Text size="xs" color="dimmed">
                  From Address for the email
                </Text>
              </Box>
              <Box>
                <ValidatedTextInput<CustomerEmailProfileFormValues>
                  label="Blind Copy"
                  placeholder="Blind Copy"
                  form={form}
                  name="blindCopy"
                  variant="filled"
                />
                <Text size="xs" color="dimmed">
                  Comma separated list of email addresses to send blind copy to
                </Text>
              </Box>
              <Box>
                <ValidatedTextInput<CustomerEmailProfileFormValues>
                  label="Read Recept To"
                  placeholder="Read Recept To"
                  form={form}
                  name="readReceiptTo"
                  variant="filled"
                />
                <Text size="xs" color="dimmed">
                  Comma separated list of email addresses to send read receipt
                  to
                </Text>
              </Box>
              <Box>
                <ValidatedTextInput<CustomerEmailProfileFormValues>
                  label="Attachment Name"
                  placeholder="Attachment Name"
                  form={form}
                  name="attachmentName"
                  variant="filled"
                />
                <Text size="xs" color="dimmed">
                  Attachment Name for the email
                </Text>
              </Box>
              <SwitchInput<CustomerEmailProfileFormValues>
                form={form}
                name="readReceipt"
                label="Read Receipt"
                description="Request a read receipt for the email"
                variant="filled"
                mt="xl"
              />
            </SimpleGrid>
            <Group position="right" mt="md">
              <Button
                type="submit"
                className={classes.control}
                loading={loading}
              >
                Submit
              </Button>
            </Group>
          </Box>
        </form>
      </Box>
    </Card>
  );
}
