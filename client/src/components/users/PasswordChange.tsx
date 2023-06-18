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

import React, { useState } from "react";
import {
  Button,
  Col,
  createStyles,
  Divider,
  Grid,
  SimpleGrid,
  Text,
} from "@mantine/core";
import axios from "@/lib/AxiosConfig";
import { useForm, yupResolver } from "@mantine/form";
import * as Yup from "yup";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { ValidatedTextInput } from "@/components/ui/fields/TextInput";

const useStyles = createStyles((theme) => ({
  text: {
    color: theme.colorScheme === "dark" ? "white" : "black",
  },
  div: {
    padding: "5px 0px 5px 0px",
  },
  invalid: {
    backgroundColor:
      theme.colorScheme === "dark"
        ? theme.fn.rgba(theme.colors.red[8], 0.15)
        : theme.colors.red[0],
  },
  icon: {
    color: theme.colors.red[theme.colorScheme === "dark" ? 7 : 6],
  },
}));

export const PasswordChangeForm: React.FC = () => {
  const [isEditing, setIsEditing] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const { classes } = useStyles();

  const startEditing = () => setIsEditing(true);
  const stopEditing = () => {
    setIsEditing(false);
    form.reset();
  };

  interface FormValues {
    old_password: string;
    new_password: string;
    confirm_password: string;
    non_field_errors?: string;
  }

  const schema = Yup.object().shape({
    old_password: Yup.string().required("Current Password is required"),
    new_password: Yup.string().required("New Password is required"),
    confirm_password: Yup.string()
      .oneOf(
        [Yup.ref("new_password"), undefined],
        "The password and its confirm are not the same"
      )
      .required("Confirm Password is required"),
  });
  const form = useForm<FormValues>({
    validate: yupResolver(schema),
    initialValues: {
      old_password: "",
      new_password: "",
      confirm_password: "",
      non_field_errors: "",
    },
  });

  const submitForm = async (values: FormValues) => {
    setLoading(true);
    try {
      const response = await axios.put("change_password/", values);

      if (response.status === 200) {
        notifications.show({
          title: "Password Changed",
          message: "Your password has been changed successfully",
          color: "green",
          withCloseButton: true,
          icon: <FontAwesomeIcon icon={faCheck} />,
        });
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
      <div className={classes.div}>
        {!isEditing ? (
          <>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                flexWrap: "wrap",
              }}
            >
              <div>
                <Text size="sm" className={classes.text} weight={700}>
                  Password
                </Text>
                <Text color="dimmed">************</Text>
              </div>
              <div
                style={{
                  marginLeft: "auto",
                }}
              >
                <Button color="gray" variant="light" onClick={startEditing}>
                  Reset Password
                </Button>
              </div>
            </div>
          </>
        ) : (
          <Grid>
            <Col w="auto">
              <form onSubmit={form.onSubmit((values) => submitForm(values))}>
                <SimpleGrid cols={3} mb={20}>
                  <ValidatedTextInput
                    label="Current Password"
                    type="password"
                    variant="filled"
                    form={form}
                    name={"old_password"}
                  />
                  <ValidatedTextInput
                    label="New Password"
                    type="password"
                    variant="filled"
                    form={form}
                    name={"new_password"}
                  />
                  <ValidatedTextInput
                    label="Confirm New Password"
                    type="password"
                    variant="filled"
                    form={form}
                    name={"confirm_password"}
                  />
                </SimpleGrid>
                <Button type="submit" color="blue" mx="xs" loading={loading}>
                  Update Password
                </Button>
                <Button type="button" onClick={stopEditing}>
                  Cancel
                </Button>
              </form>
            </Col>
          </Grid>
        )}
      </div>
      <Divider my="sm" variant="dashed" />
    </>
  );
};
