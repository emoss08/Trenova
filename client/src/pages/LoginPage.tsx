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

import { Link, useNavigate } from "react-router-dom";
import React from "react";
import {
  Anchor,
  Button,
  Checkbox,
  Container,
  Group,
  Paper,
  Text,
  Title,
} from "@mantine/core";
import { useForm, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faXmark } from "@fortawesome/pro-solid-svg-icons";
import { faLockKeyhole, faUser } from "@fortawesome/pro-duotone-svg-icons";
import { useAuthStore, useUserStore } from "@/stores/AuthStore";
import axios from "@/helpers/AxiosConfig";
import { ValidatedPasswordInput } from "@/components/common/fields/PasswordInput";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { LoginSchema } from "@/helpers/schemas/AccountsSchema";

type LoginFormValues = {
  username: string;
  password: string;
};

function LoginPage() {
  const [isAuthenticated, setIsAuthenticated] = useAuthStore(
    (state: { isAuthenticated: any; setIsAuthenticated: any }) => [
      state.isAuthenticated,
      state.setIsAuthenticated,
    ],
  );
  const [loading, setLoading] = React.useState<boolean>(false);
  const [, setUserDetails] = useUserStore.use("user");

  const form = useForm<LoginFormValues>({
    validate: yupResolver(LoginSchema),
    initialValues: {
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

  const navigate = useNavigate();
  React.useEffect((): void => {
    if (isAuthenticated) {
      const returnUrl = sessionStorage.getItem("returnUrl") || "/";
      sessionStorage.removeItem("returnUrl");
      navigate(returnUrl);
    }
  }, [isAuthenticated, navigate]);

  const login = async (values: LoginFormValues) => {
    setLoading(true);
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
        if (data.type === "validation_error") {
          data.errors.forEach((e: any) => {
            form.setFieldError(e.attr, e.detail);
            if (e.attr === "nonFieldErrors") {
              notifications.show({
                title: "Error",
                message: e.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            }
          });
        } else {
          form.setFieldError("username", error.response.data.detail);
          form.setFieldError("password", error.response.data.detail);
          notifications.show({
            title: "Error",
            message: error.response.data.detail,
            color: "red",
            withCloseButton: true,
            icon: <FontAwesomeIcon icon={faXmark} />,
            autoClose: 10_000, // 10 seconds
          });
        }
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container size={450} my={50}>
      <Title align="center">Welcome back!</Title>
      <Text color="dimmed" size="sm" align="center" mt={5}>
        Do not have an account yet?{" "}
        <Anchor size="sm" component="button">
          Create account
        </Anchor>
      </Text>

      <Paper shadow="md" p={30} mt={30} radius="md">
        <form
          onSubmit={form.onSubmit((values: LoginFormValues) => {
            login(values);
          })}
        >
          <ValidatedTextInput
            form={form}
            name="username"
            label="Username"
            placeholder="Your Username"
            withAsterisk
            variant="filled"
            icon={<FontAwesomeIcon icon={faUser} />}
          />
          <ValidatedPasswordInput
            form={form}
            name="password"
            label="Password"
            placeholder="Your password"
            mt="md"
            withAsterisk
            variant="filled"
            icon={<FontAwesomeIcon icon={faLockKeyhole} />}
          />
          <Group position="apart" mt="lg">
            <Checkbox label="Remember me" />
            <Link to="/reset-password/">
              <Anchor component="button" size="sm">
                Forgot password?
              </Anchor>
            </Link>
          </Group>
          <Button type="submit" fullWidth mt="xl" loading={loading}>
            Sign in
          </Button>
        </form>
      </Paper>
    </Container>
  );
}
export default LoginPage;
