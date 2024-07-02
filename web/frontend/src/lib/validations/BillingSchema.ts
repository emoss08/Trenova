import { StatusChoiceProps } from "@/types";
import {
  AccessorialChargeFormValues,
  BillingControlFormValues,
  ChargeTypeFormValues,
  DocumentClassificationFormValues,
} from "@/types/billing";
import {
  AutoBillingCriteriaChoicesProps,
  FuelMethodChoicesProps,
  ShipmentTransferCriteriaChoicesProps,
} from "@/utils/apps/billing";
import * as Yup from "yup";
import { ObjectSchema, StringSchema } from "yup";

Yup.addMethod<StringSchema>(
  Yup.string,
  "decimal",
  function (message = "Must be a decimal") {
    return this.test("is-decimal", message, (value) => {
      if (value) {
        return /^\d+(\.\d{1,4})?$/.test(value);
      }
      return true;
    });
  },
);

export const accessorialChargeSchema: ObjectSchema<AccessorialChargeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string()
      .max(10, "Code must be less than 10 characters.")
      .required("Code is required"),
    description: Yup.string(),
    isDetention: Yup.boolean().required("Detention is required"),
    amount: Yup.string().decimal().required("Amount is required"),
    method: Yup.string<FuelMethodChoicesProps>().required("Method is required"),
  });

export const chargeTypeSchema: ObjectSchema<ChargeTypeFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    name: Yup.string()
      .max(50, "Name must be less than 50 characters.")
      .required("Name is required"),
    description: Yup.string()
      .max(100, "Description must be less than 100 characters.")
      .notRequired(),
  });

export const documentClassSchema: ObjectSchema<DocumentClassificationFormValues> =
  Yup.object().shape({
    status: Yup.string<StatusChoiceProps>().required("Status is required"),
    code: Yup.string()
      .max(10, "Code must be less than 10 characters.")
      .required("Code is required"),
    description: Yup.string().max(
      100,
      "Description must be less than 100 characters.",
    ),
    color: Yup.string().max(100, "Color cannot be more than 100 characters"),
  });

export const billingControlSchema: ObjectSchema<BillingControlFormValues> =
  Yup.object().shape({
    removeBillingHistory: Yup.boolean().required(
      "Remove billing history is required",
    ),
    autoBillShipment: Yup.boolean().required("Auto bill shipment is required"),
    autoMarkReadyToBill: Yup.boolean().required(
      "Auto mark ready to bill is required",
    ),
    validateCustomerRates: Yup.boolean().required(
      "Validate customer rates is required",
    ),
    autoBillCriteria: Yup.string<AutoBillingCriteriaChoicesProps>().required(
      "Auto bill criteria is required",
    ),
    shipmentTransferCriteria:
      Yup.string<ShipmentTransferCriteriaChoicesProps>().required(
        "Order transfer criteria is required",
      ),
    enforceCustomerBilling: Yup.boolean().required(
      "Enforce customer billing is required",
    ),
  });
