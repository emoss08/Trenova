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
import { ModeToggle } from "@/components/theme-switcher";
import { cn } from "@/lib/utils";
import { Link, useNavigate } from "react-router-dom";
import { useAuthStore, useUserStore } from "@/stores/AuthStore";
import { useForm } from "react-hook-form";
import { yupResolver } from "@hookform/resolvers/yup";
import { userAuthSchema } from "@/lib/validations/accounts";
import axios from "@/lib/AxiosConfig";
import { Label } from "@/components/ui/label";
import { InputField, PasswordField } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";

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

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm({
    resolver: yupResolver(userAuthSchema),
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
        console.log(data);
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(login)}>
      <div className="grid gap-4 mt-5">
        <div className="grid gap-2">
          <Label htmlFor="username" className="required-label">
            Username
          </Label>
          <InputField
            id="username"
            autoCapitalize="none"
            type="text"
            autoCorrect="off"
            placeholder="Username"
            disabled={isLoading}
            error={errors?.username?.message}
            {...register("username")}
          />
        </div>
        <div className="relative grid gap-2">
          <Label htmlFor="password" className="required-label">
            Password
          </Label>
          <div className="relative">
            <PasswordField
              id="password"
              autoCapitalize="none"
              type="password"
              autoCorrect="off"
              placeholder="Password"
              disabled={isLoading}
              error={errors?.password?.message}
              {...register("password")}
            />
          </div>
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
        <Button disabled={isLoading} className="w-full my-2">
          {isLoading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Signing In...
            </>
          ) : (
            "Sign In"
          )}
        </Button>
      </div>
    </form>
  );
}

export default function LoginPage() {
  const [isAuthenticated, _] = useAuthStore(
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
      <div className="flex flex-row justify-center items-start">
        <Card className={cn("w-[420px]")}>
          <CardContent>
            <UserAuthForm />
          </CardContent>
        </Card>
      </div>
      <div className="absolute bottom-10 right-10">
        <ModeToggle />
      </div>
    </div>
  );
}
