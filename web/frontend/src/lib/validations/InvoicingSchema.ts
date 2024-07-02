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
