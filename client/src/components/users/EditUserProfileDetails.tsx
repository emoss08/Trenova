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
import {
  Card,
  createStyles,
  Flex,
  rem,
  Text,
  SimpleGrid,
  Button,
  Group,
  Divider,
} from "@mantine/core";
import { useForm, yupResolver } from "@mantine/form";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { faCheck } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import axios from "@/lib/AxiosConfig";
import { ValidatedTextInput } from "../ui/fields/TextInput";
import { StateSelect } from "../ui/fields/StateSelect";
import { CityAutoCompleteField } from "../ui/fields/CityAutoCompleteField";
import { UserSchema } from "@/utils/schema";
import { User } from "@/types/apps/accounts";

type Props = {
  user: User;
};

interface UserDetailFormValues {
  id: string;
  username: string;
  email: string;
  profile: {
    organization: string;
    first_name: string;
    last_name: string;
    address_line_1: string;
    address_line_2: string;
    city: string;
    state: string;
    zip_code: string;
    phone_number: string;
  };
}

const useStyles = createStyles((theme) => {
  const BREAKPOINT = theme.fn.smallerThan("sm");

  return {
    card: {
      width: "100%",
      maxWidth: "100%",
      height: "auto",
      "@media (max-width: 576px)": {
        height: "auto",
        maxHeight: "none",
      },
    },
    title: {
      marginBottom: `calc(${theme.spacing.xl} * 1.5)`,
      fontFamily: `Greycliff CF, ${theme.fontFamily}`,

      [BREAKPOINT]: {
        marginBottom: theme.spacing.xl,
      },
    },
    fields: {
      marginTop: rem(10),
    },
    icon: {
      marginRight: "5px",
      marginTop: "5px",
    },
    div: {
      display: "flex",
    },
    text: {
      color: theme.colorScheme === "dark" ? "white" : "black",
    },
    grid: {
      display: "flex",
    },
    control: {
      [BREAKPOINT]: {
        flex: 1,
      },
    },
    invalid: {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.fn.rgba(theme.colors.red[8], 0.15)
          : theme.colors.red[0],
    },
    invalidIcon: {
      color: theme.colors.red[theme.colorScheme === "dark" ? 7 : 6],
    },
  };
});

const EditUserProfileDetails: React.FC<Props> = ({ user }) => {
  const { classes } = useStyles();
  const [loading, setLoading] = React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: UserDetailFormValues) => axios.put(`/users/${values.id}/`, values),
    {
      onSuccess: () => {
        queryClient.invalidateQueries("user").then(() => {
          notifications.show({
            title: "Success",
            message: "User profile updated",
            color: "green",
            withCloseButton: true,
            icon: <FontAwesomeIcon icon={faCheck} />,
          });
        });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((error: any) => {
            form.setFieldError(error.attr, error.detail);
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    }
  );

  const submitForm = (values: UserDetailFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  const form = useForm<UserDetailFormValues>({
    validate: yupResolver(UserSchema),
    initialValues: {
      id: user.id,
      username: user.username,
      email: user.email,
      profile: {
        organization: user.profile?.organization || "",
        first_name: user.profile?.first_name || "",
        last_name: user.profile?.last_name || "",
        address_line_1: user.profile?.address_line_1 || "",
        address_line_2: user.profile?.address_line_2 || "",
        city: user.profile?.city || "",
        state: user.profile?.state || "",
        zip_code: user.profile?.zip_code || "",
        phone_number: user.profile?.phone_number || "",
      },
    },
  });

  return (
    <Flex>
      <Card className={classes.card} withBorder>
        <form onSubmit={form.onSubmit((values) => submitForm(values))}>
          <Text fz="xl" fw={700} className={classes.text}>
              Profile Details
          </Text>

          <Divider my={10} />

          <div className={classes.fields}>
            <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
              <ValidatedTextInput
                form={form}
                className={classes.fields}
                name="profile.first_name"
                label="First Name"
                placeholder="First Name"
                variant="filled"
                withAsterisk
              />
              <ValidatedTextInput
                form={form}
                className={classes.fields}
                name="profile.last_name"
                label="Last Name"
                placeholder="Last Name"
                variant="filled"
                withAsterisk
              />
            </SimpleGrid>
            <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
              <ValidatedTextInput
                form={form}
                className={classes.fields}
                name="profile.address_line_1"
                label="Address Line 1"
                placeholder="Address Line 1"
                variant="filled"
                withAsterisk
              />
              <ValidatedTextInput
                form={form}
                className={classes.fields}
                name="profile.address_line_2"
                label="Address Line 2"
                placeholder="Address Line 2"
                variant="filled"
              />
            </SimpleGrid>
            <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
              <CityAutoCompleteField
                form={form}
                stateSelection={form.values.profile.state}
                className={classes.fields}
                name="profile.city"
                label="City"
                placeholder="City"
                variant="filled"
                withAsterisk
              />
              <StateSelect
                label="State"
                className={classes.fields}
                placeholder="State"
                variant="filled"
                searchable
                form={form}
                name="profile.state"
              />
            </SimpleGrid>
            <ValidatedTextInput
              form={form}
              className={classes.fields}
              name="profile.zip_code"
              label="Zip Code"
              placeholder="Zip Code"
              variant="filled"
              withAsterisk
            />
            <ValidatedTextInput
              form={form}
              className={classes.fields}
              name="profile.phone_number"
              label="Phone Number"
              placeholder="Phone Number"
              variant="filled"
            />
            <Group position="right" mt="md">
              <Button
                color="white"
                type="submit"
                className={classes.control}
                loading={loading}
              >
                  Submit
              </Button>
            </Group>
          </div>
        </form>
      </Card>
    </Flex>
  );
};

export default EditUserProfileDetails;
