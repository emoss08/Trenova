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

import { Checkbox } from "@/components/common/fields/checkbox";
import { InputField, PasswordField } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import axios from "@/lib/axiosConfig";
import { userAuthSchema } from "@/lib/validations/AccountsSchema";
import { useAuthStore, useUserStore } from "@/stores/AuthStore";
import { yupResolver } from "@hookform/resolvers/yup";
import { Image } from "@unpic/react";
import React from "react";
import { useForm } from "react-hook-form";
import { Link, useNavigate } from "react-router-dom";

import { InternalLink } from "@/components/ui/link";
import trenovaLogo from "../assets/images/logo.avif";

type LoginFormValues = {
  username: string;
  password: string;
};

function AuthFooter() {
  return (
    <footer className="text-muted-foreground absolute bottom-10 w-full text-center">
      <div className="flex items-center justify-center gap-2">
        <p className="text-xs">&copy; 2024 Trenova. All rights reserved.</p>
        <span className="text-xs">|</span>
        <InternalLink to="/terms" className="text-xs hover:underline">
          Terms of Service
        </InternalLink>
        <span className="text-xs">|</span>
        <InternalLink to="/privacy" className="text-xs hover:underline">
          Privacy Policy
        </InternalLink>
      </div>
    </footer>
  );
}

function UserAuthForm() {
  const [, setIsAuthenticated] = useAuthStore(
    (state: { isAuthenticated: boolean; setIsAuthenticated: any }) => [
      state.isAuthenticated,
      state.setIsAuthenticated,
    ],
  );
  const [, setUserDetails] = useUserStore.use("user");
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit, setError } = useForm<LoginFormValues>({
    resolver: yupResolver(userAuthSchema),
    defaultValues: {
      username: "",
      password: "",
    },
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
    setIsSubmitting(true);
    try {
      const response = await axios.post("auth/login/", {
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
          setError(error.attr, {
            type: error.code,
            message: error.detail,
          });
        });
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit(login)}>
      <div className="mt-5 grid gap-4">
        <div className="grid gap-1">
          <InputField
            name="username"
            rules={{ required: true }}
            control={control}
            label="Username"
            placeholder="Username"
            autoCapitalize="none"
            autoCorrect="off"
            autoComplete="username"
          />
        </div>
        <div className="relative grid gap-1">
          <PasswordField
            name="password"
            rules={{ required: true }}
            control={control}
            label="Password"
            autoCapitalize="none"
            autoCorrect="off"
            autoComplete="current-password"
            placeholder="Password"
          />
        </div>
        <div className="mt-2 flex items-center justify-between">
          <div>
            <div className="flex items-center gap-x-1">
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
          className="my-2 w-full"
          isLoading={isSubmitting}
          loadingText="Logging In..."
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
    <div className="relative min-h-screen pt-20">
      <div className="flex flex-col items-center justify-start space-y-4">
        <Image
          src={trenovaLogo}
          layout="constrained"
          className="mb-5 w-[200px]"
          alt="trenova-logo"
          width={75}
          height={75}
        />
        <Card className="w-[420px]">
          <CardContent className="pt-0">
            <h2 className="pb-2 text-center text-xl font-semibold tracking-tight transition-colors first:mt-5">
              Sign in to Trenova
            </h2>
            <span className="flex justify-center space-y-5 text-sm">
              Do not have an account yet?&nbsp;
              <a
                href="#"
                className="text-primary text-sm font-semibold underline underline-offset-4 hover:decoration-blue-500"
              >
                Create an Account
              </a>
            </span>
            <UserAuthForm />
          </CardContent>
        </Card>
      </div>
      <AuthFooter />
    </div>
  );
}
