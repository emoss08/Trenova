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
import { SwitchInput } from "@/components/ui/fields/SwitchInput";
import { SelectInput } from "@/components/ui/fields/SelectInput";
import { APIError } from "@/types/server";
import { ValidatedNumberInput } from "@/components/ui/fields/NumberInput";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import {
  InvoiceControl,
  InvoiceControlFormValues,
} from "@/types/apps/invoicing";
import { ValidatedTextArea } from "@/components/ui/fields/TextArea";
import { dateFormatChoices } from "@/utils/apps/invoicing";
import { ValidatedFileInput } from "@/components/ui/fields/FileInput";
import { invoiceControlSchema } from "@/utils/apps/invoicing/schema";
import { useFormStyles } from "@/styles/FormStyles";

interface Props {
  invoiceControl: InvoiceControl;
}

export function InvoiceControlForm({
  invoiceControl,
}: Props): React.ReactElement {
  const { classes } = useFormStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: InvoiceControlFormValues | FormData) =>
      axios.put(`/invoice_control/${invoiceControl.id}/`, values, {
        headers: {
          "Content-Type": "multipart/form-data",
        },
      }),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["invoiceControl"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Invoice Control updated successfully",
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

  const form = useForm<InvoiceControlFormValues>({
    validate: yupResolver(invoiceControlSchema),
    initialValues: {
      invoiceNumberPrefix: invoiceControl.invoiceNumberPrefix,
      creditMemoNumberPrefix: invoiceControl.creditMemoNumberPrefix,
      invoiceDueAfterDays: invoiceControl.invoiceDueAfterDays,
      invoiceTerms: invoiceControl.invoiceTerms || "",
      invoiceFooter: invoiceControl.invoiceFooter || "",
      invoiceLogo: invoiceControl.invoiceLogo || "",
      invoiceLogoWidth: invoiceControl.invoiceLogoWidth,
      showInvoiceDueDate: invoiceControl.showInvoiceDueDate,
      invoiceDateFormat: invoiceControl.invoiceDateFormat,
      showAmountDue: invoiceControl.showAmountDue,
      attachPdf: invoiceControl.attachPdf,
    },
  });

  const handleSubmit = (values: InvoiceControlFormValues) => {
    setLoading(true);
    const formData = new FormData();

    Object.keys(values).forEach((key) => {
      const element = values[key as keyof InvoiceControlFormValues];
      if (element instanceof File || typeof element === "string") {
        formData.append(key, element);
      } else if (typeof element === "boolean" || typeof element === "number") {
        formData.append(key, element.toString());
      }
    });

    mutation.mutate(formData);
  };

  return (
    <form onSubmit={form.onSubmit((values) => handleSubmit(values))}>
      <Box className={classes.div}>
        <Box>
          <SimpleGrid cols={3} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <ValidatedTextInput<InvoiceControlFormValues>
              form={form}
              className={classes.fields}
              name="invoiceNumberPrefix"
              label="Invoice Number Prefix"
              placeholder="Invoice Number Prefix"
              variant="filled"
              description="Define a prefix for invoice numbers."
              withAsterisk
            />
            <ValidatedTextInput<InvoiceControlFormValues>
              form={form}
              className={classes.fields}
              name="creditMemoNumberPrefix"
              label="Credit Memo Number Prefix"
              placeholder="Credit Memo Number Prefix"
              description="Define a prefix for credit note numbers."
              variant="filled"
              withAsterisk
            />
            <ValidatedNumberInput<InvoiceControlFormValues>
              form={form}
              className={classes.fields}
              name="invoiceDueAfterDays"
              label="Invoice Due After Days"
              placeholder="Invoice Due After Days"
              description="Define the number of days after which an invoice is due."
              variant="filled"
              withAsterisk
            />
          </SimpleGrid>
          <ValidatedTextArea<InvoiceControlFormValues>
            form={form}
            className={classes.fields}
            name="invoiceTerms"
            label="Invoice Terms"
            placeholder="Invoice Terms"
            description="Define the terms and conditions for invoices."
            variant="filled"
          />
          <ValidatedTextArea<InvoiceControlFormValues>
            form={form}
            className={classes.fields}
            name="invoiceFooter"
            label="Invoice Footer"
            description="Define the footer for invoices."
            placeholder="Invoice Footer"
            variant="filled"
          />
          <SimpleGrid cols={3} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
            <SelectInput<InvoiceControlFormValues>
              form={form}
              data={dateFormatChoices}
              className={classes.fields}
              name="invoiceDateFormat"
              label="Invoice Date Format"
              placeholder="Invoice Date Format"
              description="Define the date format for invoices."
              variant="filled"
              withAsterisk
            />
            <ValidatedNumberInput<InvoiceControlFormValues>
              form={form}
              className={classes.fields}
              name="invoiceLogoWidth"
              label="Invoice Logo Width"
              placeholder="Invoice Logo Width"
              description="Define the width of the invoice logo."
              variant="filled"
              withAsterisk
            />
            <ValidatedFileInput<InvoiceControlFormValues>
              form={form}
              className={classes.fields}
              name="invoiceLogo"
              label="Invoice Logo"
              placeholder="Invoice Logo"
              description="Define the logo for invoices."
              variant="filled"
              value={form.values.invoiceLogo}
              accept="image/png,image/jpeg"
            />
            <SwitchInput<InvoiceControlFormValues>
              form={form}
              className={classes.fields}
              name="showInvoiceDueDate"
              label="Show Invoice Due Date"
              description="Show the invoice due date on the invoice."
            />
            <SwitchInput<InvoiceControlFormValues>
              form={form}
              className={classes.fields}
              name="showAmountDue"
              label="Show Amount Due"
              description="Show the amount due on the invoice."
            />
            <SwitchInput<InvoiceControlFormValues>
              form={form}
              className={classes.fields}
              name="attachPdf"
              label="Attach PDF"
              description="Attach the invoice PDF to the invoice email."
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
