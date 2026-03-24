import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { authService } from "@/services/auth";
import { useAuthStore } from "@/stores/auth-store";
import { loginRequestSchema, type LoginRequest } from "@/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useNavigate } from "react-router";

export function LoginForm() {
  const navigate = useNavigate();
  const setUser = useAuthStore((state) => state.setUser);

  const { control, handleSubmit, setError } = useForm<LoginRequest>({
    resolver: zodResolver(loginRequestSchema),
    defaultValues: {
      emailAddress: "admin@trenova.app",
      password: "admin123!",
    },
  });

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: authService.login,
    setFormError: setError,
    resourceName: "Login",
    onSuccess: (data) => {
      setUser(data.user);
      void navigate("/", { replace: true });
    },
  });

  const onSubmit = (data: LoginRequest) => {
    void mutateAsync(data);
  };

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <FormGroup cols={1}>
        <FormControl cols="full">
          <InputField
            name="emailAddress"
            control={control}
            label="Email Address"
            placeholder="name@work-email.com"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl cols="full">
          <SensitiveField
            name="password"
            control={control}
            placeholder="*****"
            label="Password"
            rules={{ required: true }}
          />
        </FormControl>
        <Button
          type="submit"
          className="w-full"
          isLoading={isPending}
          loadingText="Signing in..."
        >
          Sign in
        </Button>
      </FormGroup>
    </Form>
  );
}
