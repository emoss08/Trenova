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
import { useNavigate } from "react-router-dom";
import { useAuthStore } from "@/stores/authStore";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { withFormik, FormikProps } from "formik";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { useErrorStore } from "@/stores/errorStore";
import { LoginFormValues } from "@/types/login";
import { LoginSchema } from "@/utils/schema";
import { Loader2 } from "lucide-react";
import { useUserStore } from "@/stores/userStore";
import axios from "axios";

const Login: React.FC = () => {
    const [isAuthenticated, setIsAuthenticated] = useAuthStore((state) => [
      state.isAuthenticated,
      state.setIsAuthenticated
    ]);
    const LOGIN_PAGE_BG_IMAGE_URL = import.meta.env.VITE_BACKGROUND_IMAGE_URL as string;
    const [buttonStatus, setButtonStatus] = React.useState<"idle" | "processing" | "error" | "success">("idle");
    const { errorMessages, setErrorMessages } = useErrorStore();
    const [, setUser] = useUserStore((state) => [state.user, state.setUser]);
    const [backgroundImageUrl, setBackgroundImageUrl] = React.useState<string>(LOGIN_PAGE_BG_IMAGE_URL);

    const navigate = useNavigate();
    React.useEffect((): void => {
      if (isAuthenticated) {
        navigate("/");
      }
    }, [isAuthenticated, navigate]);

    const InnerLoginForm = (props: FormikProps<LoginFormValues>) => {
      const { touched, errors, handleSubmit, handleBlur, handleChange, isSubmitting } = props;
      const renderErrorMessages = (): JSX.Element | null => {
        if (errorMessages.length > 0) {
          return (
            <div className="flex flex-col space-y-2 text-center">
              {errorMessages.map((message: any, index: any) => {
                return <p key={index} className={cn(
                  "text-sm text-rose-700"
                )}>{message}</p>;
              })}
            </div>
          );
        }
        return null;
      };

      return (
        <>
          <form onSubmit={handleSubmit}>
            <div className="grid gap-2">
              <div className="grid gap-1">

                {/* Error Messages */}
                {renderErrorMessages()}

                {/* Username Field */}
                <Label htmlFor="username">Username</Label>
                <Input
                  className={cn(
                    "flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus:bg-black focus:bg-opacity-10 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
                    errors.username && touched.username && "border-rose-700"
                  )}
                  id="username"
                  onBlur={handleBlur}
                  onChange={handleChange}
                  name="username"
                  placeholder="Username"
                  type="text" />
                {errors.username && touched.username && <p className={cn("text-sm text-rose-700")}>{errors.username}</p>}

                {/* Password Field */}
                <Label htmlFor="password">Password</Label>
                <Input
                  className={cn(
                    "flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus:bg-black focus:bg-opacity-10 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
                    errors.password && touched.password && "border-rose-700"
                  )}
                  id="password"
                  name="password"
                  onBlur={handleBlur}
                  onChange={handleChange}
                  placeholder="Password"
                  type="password" />
                {errors.password && touched.password && <p className={cn("text-sm text-rose-700")}>{errors.password}</p>}
              </div>
              <Button
                disabled={isSubmitting || buttonStatus === "processing"}
                type="submit"
                className={cn(
                  "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background border border-input h-10 py-2 px-4",
                  buttonStatus === "error" ? "bg-red-500 text-white" : buttonStatus === "success" ? "bg-green-500 text-white" : "hover:bg-accent hover:text-accent-foreground"
                )}
              >
                {buttonStatus === "processing" ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Processing...
                  </>
                ) : buttonStatus === "success" ? (
                  "Success"
                ) : buttonStatus === "error" ? (
                  "Invalid, try again."
                ) : (
                  "Sign in"
                )}
              </Button>
            </div>
          </form>
        </>
      );
    };

    const LoginForm = withFormik<{}, LoginFormValues>({

      mapPropsToValues: (): { password: string, username: string } => ({ username: "", password: "" }),

      validationSchema: LoginSchema,

      handleSubmit: async (values, { setSubmitting }): Promise<void> => {
        setButtonStatus("processing");
        try {
          const response = await axios.post("login/", {
            username: values.username,
            password: values.password
          });

          localStorage.setItem("mt_user_info", JSON.stringify(response.data));
          setButtonStatus("success");
          setIsAuthenticated(true);
          setUser(response.data);
        } catch (error: any) {
          if (error.response && error.response.status === 400) {
            const errors = error.response.data.errors;
            const messages = errors.map((error: any) => error.detail);
            setErrorMessages(messages);
            setButtonStatus("error");
          } else {
            setErrorMessages(["An error occurred, please try again later"]);
            setButtonStatus("error");
          }
        }
        setSubmitting(false);
      },
      displayName: "LoginForm"
    })(InnerLoginForm);

    return (
      <>
        <div
          className="container relative hidden h-screen flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
          <div className="relative hidden h-full flex-col bg-muted p-10 text-white dark:border-r lg:flex">
            <div className="absolute inset-0 bg-cover" style={{
              backgroundImage: "url('https://images.unsplash.com/photo-1506306460327-3164753b74c7?ixlib=rb-4.0.3&ixid=MnwxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8&auto=format&fit=crop&w=687&q=80')",
              backgroundSize: "cover",
              backgroundPosition: "center"
            }}>
            </div>
            <div className="relative z-20 flex items-center text-white text-lg font-medium">
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none"
                   stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
                   className="mr-2 h-6 w-6">
                <path d="M15 6v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3"></path>
              </svg>
              MONTA
            </div>
            <div className="relative z-20 mt-auto">
              <blockquote className="space-y-2">
                <p className="text-lg text-white">“Success is not the key to happiness. Happiness is the
                  key to success. If you love
                  what you are doing, you will be successful.”</p>
                <footer className="text-sm text-white">Albert Schweitzer</footer>
              </blockquote>
            </div>
          </div>
          <div className="lg:p-8">
            <div className="mx-auto flex w-full flex-col justify-center space-y-6 sm:w-[350px]">
              <div className="flex flex-col space-y-2 text-center">
                <h1 className="text-2xl font-semibold tracking-tight">Login to Monta</h1>
                <p className="text-sm text-muted-foreground">Built to make your business better!</p>
              </div>
              <div className="grid gap-6">
                <LoginForm />
              </div>
              <p className="px-8 text-center text-sm text-muted-foreground">By clicking continue, you agree to our <a
                className="underline underline-offset-4 hover:text-primary" href="/terms">Terms of Service</a> and <a
                className="underline underline-offset-4 hover:text-primary" href="/privacy">Privacy Policy</a>.</p></div>
          </div>
        </div>
      </>
    );
  }
;

export default Login;
