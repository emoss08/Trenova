/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { InputField } from "@/components/common/fields/input";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { InternalLink } from "@/components/ui/link";
import { ModeToggle } from "@/components/ui/theme-switcher";
import axios from "@/lib/axiosConfig";
import { cn } from "@/lib/utils";
import { resetPasswordSchema } from "@/lib/validations/AccountsSchema";
import { faLoader } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type FormValues = {
  email: string;
};

export function ResetPasswordForm() {
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
          <div className="flex flex-col space-y-1">
            <span className="font-semibold">
              Great! Password reset email sent.
            </span>
            <span className="text-xs">{response.data.message}</span>
          </div>,
        );
      }
    } catch (error: any) {
      toast.error(
        <div className="flex flex-col space-y-1">
          <span className="font-semibold">Oops! Something went wrong.</span>
          <span className="text-xs">
            We were unable to send you a password reset email.
          </span>
        </div>,
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
              <FontAwesomeIcon
                icon={faLoader}
                className="mr-2 size-4 animate-spin"
              />
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

function ResetPasswordPage() {
  return (
    <div className={"relative min-h-screen pt-28"}>
      <div className="flex flex-row items-start justify-center">
        <Card className={cn("w-[420px]")}>
          <CardContent className="pt-0">
            <h2 className="pb-2 text-center text-xl font-semibold tracking-tight transition-colors first:mt-5">
              Reset your password?
            </h2>
            <span className="flex justify-center space-y-5 text-sm">
              Remember your password?&nbsp;
              <InternalLink to="/login">Login instead</InternalLink>
            </span>
            <ResetPasswordForm />
          </CardContent>
        </Card>
      </div>
      <div className="absolute bottom-10 right-10">
        <ModeToggle />
      </div>
    </div>
  );
}
export default ResetPasswordPage;
