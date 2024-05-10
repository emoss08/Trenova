import { JobFunctionChoiceProps } from "@/lib/choices";
import { StatusChoiceProps } from "@/types";
import { JobTitleFormValues } from "@/types/accounts";
import * as yup from "yup";

export const jobTitleSchema: yup.ObjectSchema<JobTitleFormValues> = yup
  .object()
  .shape({
    status: yup.string<StatusChoiceProps>().required("Status is required"),
    name: yup.string().required("Name is required"),
    description: yup.string().notRequired(),
    jobFunction: yup
      .string<JobFunctionChoiceProps>()
      .required("Job Function is required"),
  });

/**
 * A yup object schema for validating login data.
 * @property username - A required string.
 * @property password - A required string.
 */
export const userAuthSchema = yup.object().shape({
  emailAddress: yup.string().email().required("Email is required."),
  password: yup.string().required("Password is required."),
});

export const checkUserEmailSchema = yup.object().shape({
  email: yup.string().email().required("Email is required."),
});

export const resetPasswordSchema: yup.ObjectSchema<{
  email: string;
}> = yup.object().shape({
  email: yup.string().email().required("Email is required."),
});

/**
 * A yup object schema for validating user profile data.
 * @property profile - An object that includes properties:
 *                     first_name, last_name, address_line_1, city, state, zip_code, phone_number.
 */
export const UserSchema = yup.object().shape({
  profile: yup.object().shape({
    firstName: yup.string().required("First name is required"),
    lastName: yup.string().required("Last name is required"),
    addressLine1: yup.string().required("Address Line 1 is required"),
    city: yup.string().required("City is required"),
    state: yup.string().required("State is required"),
    zipCode: yup.string().required("Zip Code is required"),
    phoneNumber: yup
      .string()
      .nullable()
      .test(
        "phone_number_format",
        "Phone number must be in the format (xxx) xxx-xxxx",
        (value) => {
          if (!value) {
            return true;
          } // if the string is null or undefined, skip the test
          const regex = /^\(?([0-9]{3})\)?[-. ]?([0-9]{3})[-. ]?([0-9]{4})$/;
          return regex.test(value); // apply the regex test if string exists
        },
      ),
  }),
});
