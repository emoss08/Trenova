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
import { Container, Paper, Text, Title, Button } from "@mantine/core";
import { useForm, yupResolver } from "@mantine/form";
import * as Yup from "yup";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import { useNavigate } from "react-router-dom";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import axios from "@/lib/AxiosConfig";

const ResetPasswordPage: React.FC = () => {
  const [loading, setLoading] = React.useState<boolean>(false);
  const navigate = useNavigate();

  interface FormValues {
    email: string;
  }

  const schema = Yup.object().shape({
    email: Yup.string()
      .email("Invalid email address")
      .required("Email address is required"),
  });

  const form = useForm<FormValues>({
    validate: yupResolver(schema),
    initialValues: {
      email: "",
    },
  });

  const submitForm = async (values: FormValues) => {
    setLoading(true);
    try {
      const response = await axios.post("/reset_password/", values);
      if (response.status === 200) {
        notifications.show({
          title: "Success",
          message: response.data.message,
          color: "green",
          withCloseButton: true,
          icon: <FontAwesomeIcon icon={faCheck} />,
        });
      }
    } catch (error: any) {
      if (error.response) {
        const { data } = error.response;
        for (const field in data) {
          if (data.hasOwnProperty(field)) {
            const message = data[field].join(" ");
            form.setFieldError(field, message);
          }
        }
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <Container size={450} my={50}>
      <Title align="center">Reset Password ?</Title>
      <Text color="dimmed" size="sm" align="center" mt={5}>
        Enter your email to reset your password.
      </Text>

      <Paper shadow="md" p={30} mt={30} radius="md">
        <form onSubmit={form.onSubmit((values) => submitForm(values))}>
          <ValidatedTextInput
            label="Email"
            placeholder="Your email"
            withAsterisk
            form={form}
            name="email"
          />
          <div style={{ textAlign: "center" }}>
            <Button type="submit" loading={loading} my={10} mx={10}>
              Submit
            </Button>
            <Button
              type="button"
              color="gray"
              variant="light"
              onClick={() => navigate("/login")}
            >
              Cancel
            </Button>
          </div>
        </form>
      </Paper>
    </Container>
  );
};
export default ResetPasswordPage;
