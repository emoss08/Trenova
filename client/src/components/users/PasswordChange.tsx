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
import { useForm, yupResolver } from "@mantine/form";
import * as Yup from "yup";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import axios from "@/lib/AxiosConfig";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";

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

export function PasswordChangeForm(): React.ReactElement {
  const [isEditing, setIsEditing] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(false);
  const { classes } = useStyles();

  const startEditing = () => setIsEditing(true);
  const stopEditing = () => {
    setIsEditing(false);
    form.reset();
  };

  interface FormValues {
    oldPassword: string;
    newPassword: string;
    confirmPassword: string;
    nonFieldErrors?: string;
  }

  const schema: Yup.ObjectSchema<Omit<FormValues, "nonFieldErrors">> =
    Yup.object().shape({
      oldPassword: Yup.string().required("Current Password is required"),
      newPassword: Yup.string().required("New Password is required"),
      confirmPassword: Yup.string()
        .oneOf(
          [Yup.ref("new_password"), undefined],
          "The password and its confirm are not the same",
        )
        .required("Confirm Password is required"),
    });
  const form = useForm<Omit<FormValues, "nonFieldErrors">>({
    validate: yupResolver(schema),
    initialValues: {
      oldPassword: "",
      newPassword: "",
      confirmPassword: "",
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
                    name="oldPassword"
                    withAsterisk
                  />
                  <ValidatedTextInput
                    label="New Password"
                    type="password"
                    variant="filled"
                    form={form}
                    name="newPassword"
                    withAsterisk
                  />
                  <ValidatedTextInput
                    label="Confirm New Password"
                    type="password"
                    variant="filled"
                    form={form}
                    name="confirmPassword"
                    withAsterisk
                  />
                </SimpleGrid>
                <Text color="dimmed" size="xs" mb={15}>
                  Password must be at least 8 character and contain symbols
                </Text>
                <Button type="submit" color="blue" mx="xs" loading={loading}>
                  Update Password
                </Button>
                <Button
                  type="button"
                  color="gray"
                  variant="light"
                  onClick={stopEditing}
                >
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
}
