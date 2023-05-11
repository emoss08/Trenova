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
import axios from "../lib/axiosConfig";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { withFormik, FormikProps } from "formik";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";

interface FormValues {
  username: string;
  password: string;
}

const LoginPage: React.FC = () => {
    const [isAuthenticated, setIsAuthenticated] = useAuthStore((state) => [
      state.isAuthenticated,
      state.setIsAuthenticated
    ]);

    const navigate = useNavigate();
    React.useEffect(() => {
      if (isAuthenticated) {
        navigate("/");
      }
    }, [isAuthenticated, navigate]);

    const InnerLoginForm = (props: FormikProps<FormValues>) => {
      const { touched, errors, status, handleSubmit, handleBlur, handleChange, isSubmitting } = props;
      const renderErrorMessages = () => {
        if (status && status.errorMessages) {
          return (
            <div className="flex flex-col space-y-2 text-center">

              {status.errorMessages.map((message: any, index: any) => {
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
                    "flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
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
                    "flex h-10 w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50",
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
                disabled={isSubmitting}
                type="submit"
                className={cn(
                  "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background border border-input hover:bg-accent hover:text-accent-foreground h-10 py-2 px-4"
                )}>
                Sign in
              </Button>
            </div>
          </form>
        </>
      );
    };

    const LoginForm = withFormik<{}, FormValues>({

      mapPropsToValues: () => ({ username: "", password: "" }),

      // Your validation logic
      validate: (values) => {
        const errors: Partial<FormValues> = {};

        if (!values.username) {
          errors.username = "Username is required.";
        }

        if (!values.password) {
          errors.password = "Password is required.";
        }

        return errors;
      },

      handleSubmit: async (values, { setStatus, setSubmitting }) => {
        try {
          const response = await axios.post("login/", {
            username: values.username,
            password: values.password
          });

          localStorage.setItem("mt_token", response.data.token);
          setIsAuthenticated(true);
        } catch (error: any) {
          if (error.response && error.response.status === 400) {
            const errors = error.response.data.errors;
            const messages = errors.map((error: any) => error.detail);
            console.log("setting error messages: ", messages);
            setStatus({ errorMessages: messages });
          } else {
            setStatus({ errorMessages: ["An error occurred, please try again later"] });
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
            }}></div>
            <div className="relative z-20 flex items-center text-white text-lg font-medium">
              <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none"
                   stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
                   className="mr-2 h-6 w-6">
                <path d="M15 6v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3"></path>
              </svg>
              Organization Name Here
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
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <span className="w-full border-t"></span>
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
                    <span className="bg-background px-2 text-muted-foreground">Or continue with</span>
                  </div>
                </div>
                <Button
                  className={cn(
                    "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background border border-input hover:bg-accent hover:text-accent-foreground h-10 py-2 px-4"
                  )}
                  type="button">
                  <svg viewBox="0 0 438.549 438.549" className="mr-2 h-4 w-4">
                    <path fill="currentColor"
                          d="M409.132 114.573c-19.608-33.596-46.205-60.194-79.798-79.8-33.598-19.607-70.277-29.408-110.063-29.408-39.781 0-76.472 9.804-110.063 29.408-33.596 19.605-60.192 46.204-79.8 79.8C9.803 148.168 0 184.854 0 224.63c0 47.78 13.94 90.745 41.827 128.906 27.884 38.164 63.906 64.572 108.063 79.227 5.14.954 8.945.283 11.419-1.996 2.475-2.282 3.711-5.14 3.711-8.562 0-.571-.049-5.708-.144-15.417a2549.81 2549.81 0 01-.144-25.406l-6.567 1.136c-4.187.767-9.469 1.092-15.846 1-6.374-.089-12.991-.757-19.842-1.999-6.854-1.231-13.229-4.086-19.13-8.559-5.898-4.473-10.085-10.328-12.56-17.556l-2.855-6.57c-1.903-4.374-4.899-9.233-8.992-14.559-4.093-5.331-8.232-8.945-12.419-10.848l-1.999-1.431c-1.332-.951-2.568-2.098-3.711-3.429-1.142-1.331-1.997-2.663-2.568-3.997-.572-1.335-.098-2.43 1.427-3.289 1.525-.859 4.281-1.276 8.28-1.276l5.708.853c3.807.763 8.516 3.042 14.133 6.851 5.614 3.806 10.229 8.754 13.846 14.842 4.38 7.806 9.657 13.754 15.846 17.847 6.184 4.093 12.419 6.136 18.699 6.136 6.28 0 11.704-.476 16.274-1.423 4.565-.952 8.848-2.383 12.847-4.285 1.713-12.758 6.377-22.559 13.988-29.41-10.848-1.14-20.601-2.857-29.264-5.14-8.658-2.286-17.605-5.996-26.835-11.14-9.235-5.137-16.896-11.516-22.985-19.126-6.09-7.614-11.088-17.61-14.987-29.979-3.901-12.374-5.852-26.648-5.852-42.826 0-23.035 7.52-42.637 22.557-58.817-7.044-17.318-6.379-36.732 1.997-58.24 5.52-1.715 13.706-.428 24.554 3.853 10.85 4.283 18.794 7.952 23.84 10.994 5.046 3.041 9.089 5.618 12.135 7.708 17.705-4.947 35.976-7.421 54.818-7.421s37.117 2.474 54.823 7.421l10.849-6.849c7.419-4.57 16.18-8.758 26.262-12.565 10.088-3.805 17.802-4.853 23.134-3.138 8.562 21.509 9.325 40.922 2.279 58.24 15.036 16.18 22.559 35.787 22.559 58.817 0 16.178-1.958 30.497-5.853 42.966-3.9 12.471-8.941 22.457-15.125 29.979-6.191 7.521-13.901 13.85-23.131 18.986-9.232 5.14-18.182 8.85-26.84 11.136-8.662 2.286-18.415 4.004-29.263 5.146 9.894 8.562 14.842 22.077 14.842 40.539v60.237c0 3.422 1.19 6.279 3.572 8.562 2.379 2.279 6.136 2.95 11.276 1.995 44.163-14.653 80.185-41.062 108.068-79.226 27.88-38.161 41.825-81.126 41.825-128.906-.01-39.771-9.818-76.454-29.414-110.049z"></path>
                  </svg>
                  Github
                </Button>
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

export default LoginPage;