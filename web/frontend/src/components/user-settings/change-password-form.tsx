import { useCustomMutation } from "@/hooks/useCustomMutation";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import * as yup from "yup";
import { PasswordField } from "../common/fields/input";
import { Button } from "../ui/button";

const checkPasswordRequirements = (password: string) => {
  return {
    isLongEnough: password.length >= 8,
    hasSpecialChar: /[!@#$%^&*()_\-+]/.test(password),
    hasUpper: /[A-Z]/.test(password),
    hasLower: /[a-z]/.test(password),
    hasNumber: /\d/.test(password),
    noSequential:
      !/(012|123|234|345|456|567|678|789|890|abc|bcd|cde|def|efg|fgh|ghi|hij|ijk|jkl|klm|lmn|mno|nop|opq|pqr|qrs|rst|stu|tuv|uvw|vwx|wxy|xyz)/i.test(
        password,
      ),
    noRepeated: !/(.)\1/.test(password),
    isFrequentlyChanged: true, // This should be handled by backend verification
  };
};
export default function ChangePasswordForm() {
  type ChangePasswordFormValues = {
    oldPassword: string;
    newPassword: string;
    confirmPassword: string;
  };

  const schema: yup.ObjectSchema<ChangePasswordFormValues> = yup
    .object()
    .shape({
      oldPassword: yup
        .string()
        .required("Please enter your current password to continue"),
      newPassword: yup.string().required("Please enter a new password"),
      confirmPassword: yup
        .string()
        .oneOf([yup.ref("newPassword"), undefined], "Passwords must match")
        .required("Please confirm your new password"),
    });

  const { handleSubmit, control, reset, watch } =
    useForm<ChangePasswordFormValues>({
      resolver: yupResolver(schema),
      defaultValues: {
        oldPassword: "",
        newPassword: "",
        confirmPassword: "",
      },
    });

  const passwordRequirements = checkPasswordRequirements(watch("newPassword"));

  const mutation = useCustomMutation<ChangePasswordFormValues>(control, {
    method: "POST",
    path: "/users/change-password",
    successMessage: "Password updated successfully.",
    reset,
    errorMessage: "Failed to update password.",
  });

  const onSubmit = (values: ChangePasswordFormValues) =>
    mutation.mutate(values);

  return (
    <>
      <div className="sticky top-0 z-20 mb-6 flex items-center gap-x-2">
        <h2 className="text-sm" id="personal-information">
          Change Password
        </h2>
        <p className="text-muted-foreground text-xs">
          Update your password to keep your account secure.
        </p>
      </div>
      <div className="flex">
        <form
          className="size-full md:col-span-2"
          onSubmit={handleSubmit(onSubmit)}
        >
          <div className="grid grid-cols-1 gap-x-6 gap-y-8 sm:max-w-xl sm:grid-cols-6">
            <div className="col-span-full">
              <PasswordField
                control={control}
                name="oldPassword"
                label="Current Password"
                rules={{ required: true }}
                placeholder="Current Password"
                description="Enter the password you are currently using. This is required to verify your identity and secure your account."
              />
            </div>

            <div className="col-span-full">
              <PasswordField
                control={control}
                name="newPassword"
                label="New Password"
                rules={{ required: true }}
                placeholder="New Password"
                description="Create a new password that you haven't previously used. Ensure it is strong and secure, ideally a mix of letters, numbers, and special characters."
              />
            </div>

            <div className="col-span-full">
              <PasswordField
                control={control}
                name="confirmPassword"
                label="Confirm Password"
                rules={{ required: true }}
                placeholder="Confirm Password"
                description="Re-enter your new password to confirm it. This helps ensure that you haven't mistyped your new password."
              />
            </div>
          </div>
          <div className="mt-8 flex">
            <Button type="submit" isLoading={mutation.isPending}>
              Update Password
            </Button>
          </div>
        </form>
        <div className=" mx-10 size-full">
          <h3 className=" mb-2 text-base font-semibold">
            Password Recommendations
          </h3>
          <p className="text-muted-foreground mb-2 text-xs font-normal">
            Please follow this guide for a strong password:
          </p>
          <ul className="text-muted-foreground ml-8 list-disc space-y-1 text-xs">
            <ul className="list-disc space-y-1 text-xs">
              <li
                className={`${
                  passwordRequirements.hasSpecialChar
                    ? "text-green-600"
                    : "text-red-600"
                }`}
              >
                At least one special character (e.g., ! @ # $ % ^ & * () _ - +)
              </li>
              <li
                className={`${
                  passwordRequirements.isLongEnough
                    ? "text-green-600"
                    : "text-red-600"
                }`}
              >
                At least 8 characters long
              </li>
              <li
                className={`${
                  passwordRequirements.hasUpper
                    ? "text-green-600"
                    : "text-red-600"
                }`}
              >
                At least one uppercase letter
              </li>
              <li
                className={`${
                  passwordRequirements.hasLower
                    ? "text-green-600"
                    : "text-red-600"
                }`}
              >
                At least one lowercase letter
              </li>
              <li
                className={`${
                  passwordRequirements.hasNumber
                    ? "text-green-600"
                    : "text-red-600"
                }`}
              >
                At least one number
              </li>
              <li
                className={`${
                  passwordRequirements.noSequential
                    ? "text-green-600"
                    : "text-red-600"
                }`}
              >
                No sequential characters
              </li>
              <li
                className={`${
                  passwordRequirements.noRepeated
                    ? "text-green-600"
                    : "text-red-600"
                }`}
              >
                No repeated characters
              </li>
            </ul>
          </ul>
        </div>
      </div>
    </>
  );
}
