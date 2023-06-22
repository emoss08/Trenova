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
import { Link, useNavigate } from "react-router-dom";
import {
  Anchor,
  Checkbox,
  Container,
  Group,
  Paper,
  Text,
  Title,
  Button,
} from "@mantine/core";
import { useForm, yupResolver } from "@mantine/form";
import { LoginFormValues } from "@/types/login";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faXmark } from "@fortawesome/pro-solid-svg-icons";
import { faLockKeyhole, faUser } from "@fortawesome/pro-duotone-svg-icons";
import { LoginSchema } from "@/utils/schema";
import { useAuthStore } from "@/stores/AuthStore";
import { getUserDetails } from "@/requests/UserRequestFactory";
import axios from "@/lib/AxiosConfig";
import { ValidatedPasswordInput } from "@/components/ui/fields/PasswordInput";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";
import { getUserCSRFToken } from "@/lib/utils";

const LoginPage: React.FC = () => {
  const [isAuthenticated, setIsAuthenticated] = useAuthStore((state) => [
    state.isAuthenticated,
    state.setIsAuthenticated,
  ]);
  const [loading, setLoading] = React.useState<boolean>(false);

  const form = useForm<LoginFormValues>({
    initialValues: {
      username: "",
      password: "",
    },

    validate: yupResolver(LoginSchema),
  });
  const csrfToken = getUserCSRFToken();

  const navigate = useNavigate();
  React.useEffect((): void => {
    if (isAuthenticated) {
      navigate("/");
    }
  }, [isAuthenticated, navigate]);

  const login = async (values: LoginFormValues) => {
    setLoading(true);
    try {
      const response = await axios.post(
        "/login/",
        {
          username: values.username,
          password: values.password,
        },
        {
          headers: {
            "X-CSRFToken": csrfToken,
          },
          withCredentials: true,
        }
      );

      if (response.status === 200) {
        sessionStorage.setItem("mt_user_id", response.data.user_id);
        sessionStorage.setItem(
          "mt_organization_id",
          response.data.organization_id
        );

        const userInfo = await getUserDetails(response.data.user_id);
        sessionStorage.setItem(
          "mt_user_permissions",
          JSON.stringify(userInfo.user_permissions)
        );
        sessionStorage.setItem(
          "mt_user_groups",
          JSON.stringify(userInfo.groups)
        );
        sessionStorage.setItem("mt_is_admin", userInfo.is_staff.toString());
        setIsAuthenticated(true);
      }
    } catch (error: any) {
      if (error.response) {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((error: any) => {
            form.setFieldError(error.attr, error.detail);
            if (error.attr === "non_field_errors") {
              notifications.show({
                title: "Error",
                message: error.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            }
          });
        }
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Container size={450} my={50}>
        <Title align="center">Welcome back!</Title>
        <Text color="dimmed" size="sm" align="center" mt={5}>
          Do not have an account yet?{" "}
          <Anchor size="sm" component="button">
            Create account
          </Anchor>
        </Text>

        <Paper withBorder shadow="md" p={30} mt={30} radius="md">
          <form
            onSubmit={form.onSubmit((values: LoginFormValues) => {
              login(values).then(() => {});
            })}
          >
            <ValidatedTextInput
              form={form}
              name="username"
              label="Username"
              placeholder="Your Username"
              withAsterisk
              icon={<FontAwesomeIcon icon={faUser} />}
            />
            <ValidatedPasswordInput
              form={form}
              name="password"
              label="Password"
              placeholder="Your password"
              mt="md"
              withAsterisk
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
    </>
  );
};
export default LoginPage;
