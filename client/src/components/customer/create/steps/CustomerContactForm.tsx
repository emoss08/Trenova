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
import { Box, Button, Divider, Group, useMantineTheme } from "@mantine/core";
import { CreateCustomerFormValues } from "@/types/customer";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { SwitchInput } from "@/components/common/fields/SwitchInput";

export function CustomerContactForm({
  form,
}: {
  form: UseFormReturnType<CreateCustomerFormValues>;
}) {
  const theme = useMantineTheme();

  const fields = form.values.customerContacts?.map((item, index) => {
    return (
      <>
        <Group mt="xs" key={`customer-contact-${index}`}>
          <ValidatedTextInput<CreateCustomerFormValues>
            label="Name"
            description="Name of the contact"
            form={form}
            name={`customerContacts.${index}.name`}
            sx={{ flex: 1 }}
            placeholder="Enter Name"
            withAsterisk
          />
          <ValidatedTextInput<CreateCustomerFormValues>
            label="Email Address"
            description="Email address of the contact"
            form={form}
            name={`customerContacts.${index}.email`}
            sx={{ flex: 1 }}
            placeholder="Enter Email Address"
            withAsterisk
          />
          <ValidatedTextInput<CreateCustomerFormValues>
            form={form}
            label="Title"
            description="Current job title of the contact"
            name={`customerContacts.${index}.title`}
            sx={{ flex: 1 }}
            placeholder="Enter Title"
            withAsterisk
          />
          <ValidatedTextInput<CreateCustomerFormValues>
            form={form}
            description="Phone number for the contact"
            label="Phone Number"
            name={`customerContacts.${index}.phone`}
            sx={{ flex: 1 }}
            placeholder="Enter Phone Number"
          />
        </Group>
        <Group spacing="xl">
          <SwitchInput<CreateCustomerFormValues>
            form={form}
            label="Is Active"
            name={`customerContacts.${index}.isActive`}
            description="Is customer currently active?"
            defaultChecked
          />
          <SwitchInput<CreateCustomerFormValues>
            form={form}
            label="Is Payable Contact"
            name={`customerContacts.${index}.isPayableContact`}
            description="Will contact be used when invoicing customer?"
          />
          <Button
            mt={40}
            variant="subtle"
            style={{
              color:
                theme.colorScheme === "dark"
                  ? theme.colors.gray[0]
                  : theme.colors.dark[9],
              backgroundColor: "transparent",
            }}
            size="sm"
            compact
            onClick={() => form.removeListItem("customerContacts", index)}
          >
            Remove Customer Contact
          </Button>
        </Group>
        <Divider variant="dashed" mt={20} />
      </>
    );
  });

  return (
    <Box>
      {fields}
      <Button
        variant="subtle"
        style={{
          color:
            theme.colorScheme === "dark"
              ? theme.colors.gray[0]
              : theme.colors.dark[9],
          backgroundColor: "transparent",
        }}
        size="sm"
        compact
        mt={20}
        onClick={() =>
          form.insertListItem("customerContacts", {
            name: "",
            email: "",
            title: "",
            phone: "",
            isActive: true,
            isPayableContact: false,
          })
        }
      >
        Add Customer Contact
      </Button>
    </Box>
  );
}
