/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

import { InvoiceControlFormValues } from "@/types/invoicing";
import * as Yup from "yup";
import { ObjectSchema } from "yup";
import { DateFormatChoiceProps } from "../choices";

const MAX_FILE_SIZE = 1024000; // 1MB

const validFileExtensions: any = {
  image: ["jpg", "gif", "png", "jpeg", "svg", "webp"],
};

function isValidFileType(fileName: any, fileType: any) {
  return (
    fileName &&
    validFileExtensions[fileType].indexOf(fileName.split(".").pop()) > -1
  );
}

export const invoiceControlSchema: ObjectSchema<InvoiceControlFormValues> =
  Yup.object().shape({
    invoiceNumberPrefix: Yup.string().required(
      "Invoice Number Prefix is required",
    ),
    creditMemoNumberPrefix: Yup.string().required(
      "Credit Memo Number Prefix is required",
    ),
    invoiceDueAfterDays: Yup.number().required(
      "Invoice Due After Days is required",
    ),
    invoiceTerms: Yup.string(),
    invoiceFooter: Yup.string(),
    invoiceLogoUrl: Yup.mixed()
      .test("is-valid-type", "Not a valid image type", (value: any) => {
        if (!value) return true;
        return isValidFileType(value && value.name.toLowerCase(), "image");
      })
      .test("is-valid-size", "Max allowed size is 100KB", (value: any) => {
        if (!value) return true;
        return value && value.size <= MAX_FILE_SIZE;
      })
      .notRequired(), // File Upload field
    invoiceLogoWidth: Yup.number().required("Invoice Logo Width is required"),
    showInvoiceDueDate: Yup.boolean().required(
      "Show Invoice Due Date is required",
    ),
    invoiceDateFormat: Yup.string<DateFormatChoiceProps>().required(
      "Invoice Date Format is required",
    ),
    showAmountDue: Yup.boolean().required("Show Amount Due is required"),
    attachPdf: Yup.boolean().required("Attach PDF is required"),
  });
