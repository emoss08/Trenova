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
import { UseFormReturnType } from "@mantine/form";
import { Box, Divider, SimpleGrid, Text } from "@mantine/core";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { CreateCustomerFormValues } from "@/types/customer";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { SwitchInput } from "@/components/common/fields/SwitchInput";

export function CustomerEmailProfileForm({
  form,
}: {
  form: UseFormReturnType<CreateCustomerFormValues>;
}): React.ReactElement {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <Divider my={10} />
      <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <Box>
          <ValidatedTextInput<CreateCustomerFormValues>
            label="Subject"
            placeholder="Subject"
            form={form}
            name="emailProfile.subject"
            maxLength={100}
            variant="filled"
          />
          <Text size="xs" color="dimmed">
            Subject for the email
          </Text>
        </Box>
        <Box>
          <ValidatedTextInput<CreateCustomerFormValues>
            label="Comment"
            placeholder="Comment"
            form={form}
            name="emailProfile.comment"
            maxLength={100}
            variant="filled"
          />
          <Text size="xs" color="dimmed">
            Comment for the email
          </Text>
        </Box>
        <Box>
          <ValidatedTextInput<CreateCustomerFormValues>
            label="From Address"
            placeholder="From Address"
            form={form}
            name="emailProfile.fromAddress"
            variant="filled"
          />
          <Text size="xs" color="dimmed">
            From Address for the email
          </Text>
        </Box>
        <Box>
          <ValidatedTextInput<CreateCustomerFormValues>
            label="Blind Copy"
            placeholder="Blind Copy"
            form={form}
            name="emailProfile.blindCopy"
            variant="filled"
          />
          <Text size="xs" color="dimmed">
            Comma separated list of email addresses to send blind copy to
          </Text>
        </Box>
        <Box>
          <ValidatedTextInput<CreateCustomerFormValues>
            label="Read Recept To"
            placeholder="Read Recept To"
            form={form}
            name="emailProfile.readReceiptTo"
            variant="filled"
          />
          <Text size="xs" color="dimmed">
            Comma separated list of email addresses to send read receipt to
          </Text>
        </Box>
        <Box>
          <ValidatedTextInput<CreateCustomerFormValues>
            label="Attachment Name"
            placeholder="Attachment Name"
            form={form}
            name="emailProfile.attachmentName"
            variant="filled"
          />
          <Text size="xs" color="dimmed">
            Attachment Name for the email
          </Text>
        </Box>
        <SwitchInput<CreateCustomerFormValues>
          form={form}
          name="emailProfile.readReceipt"
          label="Read Receipt"
          description="Request a read receipt for the email"
          variant="filled"
          mt="xl"
        />
      </SimpleGrid>
    </Box>
  );
}
