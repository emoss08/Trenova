import { PasswordField } from "@/components/fields/sensitive-input-field";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { Label } from "@/components/ui/label";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { login } from "@/services/auth";
import { useAuthActions } from "@/stores/user-store";
import { APIError } from "@/types/errors";
import { faLock } from "@fortawesome/pro-regular-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { useForm } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";
import { toast } from "sonner";
import { z } from "zod";

const loginSchema = z.object({
  emailAddress: z
    .string()
    .min(1, "Email is required")
    .email("Invalid email address"),
  password: z.string().min(1, "Password is required"),
  rememberMe: z.optional(z.boolean()),
});

type LoginFormValues = z.infer<typeof loginSchema>;

type LoginFormProps = {
  email: string;
  onForgotPassword: () => void;
};

export function LoginForm({ email, onForgotPassword }: LoginFormProps) {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { setUser } = useAuthActions();

  const { mutateAsync } = useMutation({
    mutationFn: async (values: LoginFormValues) => {
      const response = await login(values);
      return response.data;
    },
    onSuccess: (data) => {
      setUser(data.user);

      // Redirect to the original destination or dashboard
      const from = searchParams.get("from") || "/";
      navigate(from, { replace: true });
    },
    onError: (error: APIError) => {
      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as keyof LoginFormValues, {
            message: fieldError.reason,
          });
        });
      } else {
        toast.error(error.message || "Failed to sign in");
      }
    },
  });

  const {
    control,
    handleSubmit,
    setError,
    formState: { isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      emailAddress: email,
      password: "",
      rememberMe: false,
    },
  });

  const onSubmit = useCallback(
    async (values: LoginFormValues) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        (event.ctrlKey || event.metaKey) &&
        event.key === "Enter" &&
        !isSubmitting
      ) {
        event.preventDefault();
        handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isSubmitting, handleSubmit, onSubmit]);

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <FormGroup cols={1}>
        <FormControl className="min-h-[2.5em]">
          <div className="flex flex-col gap-1">
            <Label>Email address</Label>
            <p className="text-sm text-muted-foreground">{email}</p>
          </div>
        </FormControl>

        <FormControl className="min-h-[4em]">
          <PasswordField
            onPasswordReset={onForgotPassword}
            icon={<Icon icon={faLock} className="size-3.5" />}
            control={control}
            rules={{ required: true }}
            name="password"
            label="Password"
            placeholder="Password"
          />
        </FormControl>
        <TooltipProvider>
          <Tooltip delayDuration={400}>
            <TooltipTrigger asChild>
              <Button
                type="submit"
                isLoading={isSubmitting}
                disabled={isSubmitting}
              >
                Sign In
              </Button>
            </TooltipTrigger>
            <TooltipContent
              side="bottom"
              className="flex items-center gap-2 text-xs"
            >
              <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-foreground">
                Ctrl
              </kbd>
              <kbd className="-me-1 inline-flex h-5 max-h-full items-center rounded bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-foreground z-[100]">
                Enter
              </kbd>
              <p>to sign in</p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </FormGroup>
    </Form>
  );
}
