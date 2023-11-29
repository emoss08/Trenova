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

import { ModeToggle } from "@/components/ui/theme-switcher";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/common/fields/checkbox";
import { InputField } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import axios from "@/lib/axiosConfig";
import { cn } from "@/lib/utils";
import { userAuthSchema } from "@/lib/validations/accounts";
import { useAuthStore, useUserStore } from "@/stores/AuthStore";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router-dom";

type LoginFormValues = {
  username: string;
  password: string;
};

function UserAuthForm() {
  const [, setIsAuthenticated] = useAuthStore(
    (state: { isAuthenticated: any; setIsAuthenticated: any }) => [
      state.isAuthenticated,
      state.setIsAuthenticated,
    ],
  );
  const [, setUserDetails] = useUserStore.use("user");
  const [isLoading, setIsLoading] = React.useState<boolean>(false);

  const { control, handleSubmit, setError } = useForm<LoginFormValues>({
    resolver: yupResolver<LoginFormValues>(userAuthSchema),
  });

  const fetchUserDetails = async () => {
    try {
      const response = await axios.get("/me/", {
        withCredentials: true,
      });

      if (response.status === 200) {
        setUserDetails(response.data);
        setIsAuthenticated(true);
      }
    } catch (error) {
      setIsAuthenticated(false);
    }
  };

  const login = async (values: LoginFormValues) => {
    setIsLoading(true);
    try {
      const response = await axios.post("/login/", {
        username: values.username,
        password: values.password,
      });

      if (response.status === 200) {
        await fetchUserDetails();
        setIsAuthenticated(true);
      }
    } catch (error: any) {
      if (error.response) {
        const { data } = error.response;
        data.errors.forEach((error: any) => {
          setError("username", {
            type: error.code,
            message: error.detail,
          });
          setError("password", {
            type: error.code,
            message: error.detail,
          });
        });
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(login)}>
      <div className="grid gap-4 mt-5">
        <div className="grid gap-1">
          <InputField
            name="username"
            rules={{ required: true }}
            control={control}
            label="Username"
            id="username"
            autoCapitalize="none"
            autoCorrect="off"
            type="text"
            placeholder="Username"
            autoComplete="username"
            disabled={isLoading}
          />
        </div>
        <div className="grid gap-1">
          <InputField
            name="password"
            rules={{ required: true }}
            control={control}
            label="Password"
            id="password"
            autoCapitalize="none"
            type="password"
            autoComplete="current-password"
            autoCorrect="off"
            placeholder="Password"
            disabled={isLoading}
          />
        </div>
        <div className="flex items-center justify-between mt-2">
          <div>
            <div className="flex items-center space-x-2">
              <Checkbox id="remember-me" />
              <Label htmlFor="remember-me">Remember Me</Label>
            </div>
          </div>
          <div>
            <Link to="/reset-password" className="text-sm hover:underline">
              Forgot Password?
            </Link>
          </div>
        </div>
        <Button
          className="w-full my-2"
          isLoading={isLoading}
          loadingText="Signing In..."
        >
          Continue
        </Button>
      </div>
    </form>
  );
}

export default function LoginPage() {
  const [isAuthenticated] = useAuthStore(
    (state: { isAuthenticated: any; setIsAuthenticated: any }) => [
      state.isAuthenticated,
      state.setIsAuthenticated,
    ],
  );

  const navigate = useNavigate();
  React.useEffect((): void => {
    if (isAuthenticated) {
      const returnUrl = sessionStorage.getItem("returnUrl") || "/";
      sessionStorage.removeItem("returnUrl");
      navigate(returnUrl);
    }
  }, [isAuthenticated, navigate]);

  return (
    <div className={"relative min-h-screen pt-28"}>
      <h2 className="mt-10 text-center pb-2 text-3xl font-semibold tracking-tight transition-colors first:mt-0">
        Welcome Back
      </h2>
      <p className="mb-5 text-center leading-7">
        Do not have an account yet?&nbsp;
        <a
          href="#"
          className="font-medium text-primary underline underline-offset-4"
        >
          Create an Account
        </a>
      </p>
      <div className="flex flex-col items-center justify-start space-y-4">
        {/* Adjusted here */}
        <Card className={cn("w-[420px] shadow-sm")}>
          <CardContent className="pt-0">
            <UserAuthForm />
          </CardContent>
        </Card>
        <p className="px-8 text-center text-sm text-muted-foreground w-[350px]">
          By clicking continue, you agree to our&nbsp;
          <a
            className="underline underline-offset-4 hover:text-primary"
            href="/terms"
          >
            Terms of Service
          </a>
          &nbsp; and&nbsp;
          <a
            className="underline underline-offset-4 hover:text-primary"
            href="/privacy"
          >
            Privacy Policy
          </a>
          .
        </p>
      </div>
      <div className="absolute bottom-10 right-10">
        <ModeToggle />
      </div>
    </div>
  );
}
