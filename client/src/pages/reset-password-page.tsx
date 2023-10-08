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
import { Card, CardContent } from "@/components/ui/card";
import { cn } from "@/lib/utils";
import { ModeToggle } from "@/components/theme-switcher";
import { useAuthStore } from "@/stores/AuthStore";
import { useForm } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import { resetPasswordSchema } from "@/lib/validations/accounts";
import axios from "@/lib/AxiosConfig";
import { Label } from "@/components/ui/label";
import { InputField } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";
import { Link } from "react-router-dom";
import { useToast } from "@/components/ui/use-toast";
import { ToastAction } from "@/components/ui/toast";

type FormValues = {
  email: string;
};

export function ResetPasswordForm() {
  const { toast } = useToast();
  const [, setIsAuthenticated] = useAuthStore(
    (state: { isAuthenticated: any; setIsAuthenticated: any }) => [
      state.isAuthenticated,
      state.setIsAuthenticated,
    ],
  );
  const [isLoading, setIsLoading] = React.useState<boolean>(false);

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: yupResolver(resetPasswordSchema),
  });

  const submitForm = async (values: FormValues) => {
    setIsLoading(true);
    try {
      const response = await axios.post("/reset_password/", values);
      if (response.status === 200) {
        toast({
          title: "Email Sent",
          description: "Please check your email for the reset link.",
        });
      }
    } catch (error: any) {
      console.info("error", error);
      toast({
        variant: "destructive",
        title: "Uh oh! Something went wrong.",
        description: "There was a problem with your request.",
        action: <ToastAction altText="Try again">Try again</ToastAction>,
      });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(submitForm)}>
      <div className="grid gap-4 mt-5">
        <div className="grid gap-2">
          <Label htmlFor="email" className="required-label">
            Email
          </Label>
          <InputField
            id="email"
            autoCapitalize="none"
            type="email"
            autoCorrect="off"
            placeholder="Email Address"
            disabled={isLoading}
            error={errors?.email?.message}
            {...register("email")}
          />
        </div>
        <Button disabled={isLoading} className="w-full my-2">
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
      <h2 className="mt-10 text-center pb-2 text-3xl font-semibold tracking-tight transition-colors first:mt-0">
        Reset your password?
      </h2>
      <p className="mb-5 text-center leading-7">
        Remember your password?&nbsp;
        <Link
          to="/login"
          className="font-medium text-primary underline underline-offset-4"
        >
          Login instead
        </Link>
      </p>
      <div className="flex flex-row justify-center items-start">
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
