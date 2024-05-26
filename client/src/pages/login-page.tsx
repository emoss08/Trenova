import { Checkbox } from "@/components/common/fields/checkbox";
import { InputField, PasswordField } from "@/components/common/fields/input";
import { Label } from "@/components/common/fields/label";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import axios from "@/lib/axiosConfig";
import {
  checkUserEmailSchema,
  userAuthSchema,
} from "@/lib/validations/AccountsSchema";
import { useAuthStore, useUserStore } from "@/stores/AuthStore";
import { yupResolver } from "@hookform/resolvers/yup";
import { Image } from "@unpic/react";
import React from "react";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router-dom";

import { InternalLink } from "@/components/ui/link";
import { checkUserEmail } from "@/services/AccountRequestService";
import { HTTP_200_OK } from "@/types/server";
import { useMutation } from "@tanstack/react-query";
import trenovaLogo from "../assets/images/logo.avif";

type LoginFormValues = {
  emailAddress: string;
  password: string;
};

type CheckEmailValues = {
  email: string;
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

function CheckEmailForm({
  onEmailVerified,
}: {
  onEmailVerified: (email: string) => void;
}) {
  const mutation = useMutation({
    mutationFn: (values: CheckEmailValues) => {
      return checkUserEmail(values.email);
    },
  });

  const { control, handleSubmit, setError } = useForm<CheckEmailValues>({
    resolver: yupResolver(checkUserEmailSchema),
    defaultValues: {
      email: "",
    },
  });

  const onSubmit = async (values: CheckEmailValues) => {
    try {
      const data = await mutation.mutateAsync(values);

      if (data.accountStatus === "I") {
        setError("email", {
          type: "inactive",
          message: "Your account is inactive. Please contact support.",
        });
        return;
      }

      if (data.exists) {
        onEmailVerified(values.email);
      } else {
        setError("email", {
          type: "not-found",
          message: data.message,
        });
      }
    } catch (error: any) {
      if (error.response) {
        const { data } = error.response;
        setError("email", {
          type: data.code,
          message: data.detail,
        });
      }
    } finally {
      mutation.reset();
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <div className="mt-5 grid gap-4">
        <div className="grid gap-1">
          <InputField
            name="email"
            rules={{ required: true }}
            control={control}
            label="Email Address"
            placeholder="Email Address"
            autoCapitalize="none"
            autoCorrect="off"
            autoComplete="email"
            type="email"
          />
        </div>

        <Button
          className="my-2 w-full"
          isLoading={mutation.isPending}
          disabled={mutation.isPending}
          loadingText="Checking Email..."
        >
          Continue
        </Button>
      </div>
    </form>
  );
}

function UserAuthForm({ initialEmail }: { initialEmail: string }) {
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
      emailAddress: initialEmail,
      password: "",
    },
  });

  const fetchUserDetails = async () => {
    try {
      const response = await axios.get("/users/me/", {
        withCredentials: true,
      });

      if (response.status === HTTP_200_OK) {
        localStorage.setItem("trenova-user-id", response.data.id); // Persist user ID to localStorage
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
        emailAddress: values.emailAddress,
        password: values.password,
      });
      if (response.status === HTTP_200_OK) {
        await fetchUserDetails();
        setIsAuthenticated(true);
      }
    } catch (error: any) {
      if (error.response) {
        const { data } = error.response;
        data.errors.forEach((error: any) => {
          console.log(`[Trenova ${error.code}]: ${error.detail}`);
          setError(error.attr, { type: error.code, message: error.detail });
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
            name="emailAddress"
            rules={{ required: true }}
            control={control}
            label="Email Address"
            placeholder="Email Address"
            autoCapitalize="none"
            autoCorrect="off"
            autoComplete="emailAddress"
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
            <InternalLink className="text-sm" to="/reset-password">
              Forgot Password?
            </InternalLink>
          </div>
        </div>
        <Button
          className="my-2 w-full"
          isLoading={isSubmitting}
          disabled={isSubmitting}
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

  const [showLoginForm, setShowLoginForm] = React.useState<boolean>(false);
  const [verifiedEmail, setVerifiedEmail] = React.useState<string>("");

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
              <InternalLink to="#">Create an Account</InternalLink>
            </span>
            {showLoginForm ? (
              <UserAuthForm initialEmail={verifiedEmail} />
            ) : (
              <CheckEmailForm
                onEmailVerified={(email) => {
                  setVerifiedEmail(email);
                  setShowLoginForm(true);
                }}
              />
            )}
          </CardContent>
        </Card>
      </div>
      <AuthFooter />
    </div>
  );
}
