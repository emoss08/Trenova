/*
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

import { InputField } from "@/components/common/fields/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { ModeToggle } from "@/components/ui/theme-switcher";
import axios from "@/lib/axiosConfig";
import { TOAST_STYLE } from "@/lib/constants";
import { cn } from "@/lib/utils";
import { resetPasswordSchema } from "@/lib/validations/AccountsSchema";
import { yupResolver } from "@hookform/resolvers/yup";
import { Loader2 } from "lucide-react";
import React from "react";
import { useForm } from "react-hook-form";
import toast from "react-hot-toast";
import { Link } from "react-router-dom";

type FormValues = {
  email: string;
};

export function ResetPasswordForm() {
  // const { toast } = useToast();
  const [isLoading, setIsLoading] = React.useState<boolean>(false);

  const { control, handleSubmit } = useForm({
    resolver: yupResolver(resetPasswordSchema),
  });

  const submitForm = async (values: FormValues) => {
    setIsLoading(true);
    try {
      const response = await axios.post("/reset_password/", values);
      if (response.status === 200) {
        toast.success(
          () => (
            <div className="flex flex-col space-y-1">
              <span className="font-semibold">
                Great! Password reset email sent.
              </span>
              <span className="text-xs">{response.data.message}</span>
            </div>
          ),
          {
            duration: 4000,
            id: "notification-toast",
            style: TOAST_STYLE,
            ariaProps: {
              role: "status",
              "aria-live": "polite",
            },
          },
        );
      }
    } catch (error: any) {
      console.info("error", error);
      toast.error(
        () => (
          <div className="flex flex-col space-y-1">
            <span className="font-semibold">Oops! Something went wrong.</span>
            <span className="text-xs">
              We were unable to send you a password reset email.
            </span>
          </div>
        ),
        {
          duration: 4000,
          id: "notification-toast",
          style: TOAST_STYLE,
          ariaProps: {
            role: "status",
            "aria-live": "polite",
          },
        },
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(submitForm)}>
      <div className="mt-5 grid gap-4">
        <div className="grid gap-2">
          <InputField
            name="email"
            control={control}
            id="email"
            autoCapitalize="none"
            type="email"
            autoCorrect="off"
            placeholder="Email Address"
            disabled={isLoading}
          />
        </div>
        <Button disabled={isLoading} className="my-2 w-full">
          {isLoading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Sending Email
            </>
          ) : (
            "Reset Password"
          )}
        </Button>
      </div>
    </form>
  );
}

const ResetPasswordPage: React.FC = () => {
  // const [loading, setLoading] = React.useState<boolean>(false);
  // const navigate = useNavigate();
  //
  // interface FormValues {
  //   email: string;
  // }
  //
  // const schema = Yup.object().shape({
  //   email: Yup.string()
  //     .email("Invalid email address")
  //     .required("Email address is required"),
  // });

  // const form = useForm<FormValues>({
  //   validate: yupResolver(schema),
  //   initialValues: {
  //     email: "",
  //   },
  // });

  return (
    <div className={"relative min-h-screen pt-28"}>
      <h2 className="mt-10 pb-2 text-center text-3xl font-semibold tracking-tight transition-colors first:mt-0">
        Reset your password?
      </h2>
      <p className="mb-5 text-center leading-7">
        Remember your password?&nbsp;
        <Link
          to="/login"
          className="text-primary font-medium underline underline-offset-4"
        >
          Login instead
        </Link>
      </p>
      <div className="flex flex-row items-start justify-center">
        <Card className={cn("w-[420px]")}>
          <CardContent>
            <ResetPasswordForm />
          </CardContent>
        </Card>
      </div>
      <div className="absolute bottom-10 right-10">
        <ModeToggle />
      </div>
    </div>
  );
};
export default ResetPasswordPage;
