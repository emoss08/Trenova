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
import { User } from "@/types/user";
import React from "react";
import {
  Card,
  createStyles,
  Flex,
  rem,
  Text,
  SimpleGrid,
  TextInput,
  Button,
  Group,
  Divider,
} from "@mantine/core";
import { useForm } from "@mantine/form";
import { StateSelect } from "../ui/StateSelect";

type Props = {
  user: User;
};

interface FormValues {
  profile: {
    first_name: string;
    last_name: string;
    address_line_1: string;
    address_line_2?: string;
    city: string;
    state: string;
    zip_code: string;
    phone_number?: string;
    profile_picture: string;
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
    form: {
      boxSizing: "border-box",
      flex: 1,
      padding: theme.spacing.xl,
      paddingLeft: `calc(${theme.spacing.xl} * 2)`,
      borderLeft: 0,

      [BREAKPOINT]: {
        padding: theme.spacing.md,
        paddingLeft: theme.spacing.md,
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
      marginTop: rem(-12),
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
    field: {
      marginTop: theme.spacing.md,
      // overwrite input field BackgroundColor
      "& input": {
        backgroundColor: "rgba(0,0,0,0.10)",
      },
      // overwrite input field textColor to be more white on dark background, but darker on light background
      "& input::placeholder": {
        color:
          theme.colorScheme === "dark"
            ? "rgba(255,255,255,0.50)"
            : "rgb(101,101,101)",
      },
    },
  };
});

const EditUserProfileDetails: React.FC<Props> = ({ user }) => {
  const { classes } = useStyles();

  const form = useForm<FormValues>({
    initialValues: {
      profile: {
        first_name: user.profile?.first_name || "",
        last_name: user.profile?.last_name || "",
        address_line_1: user.profile?.address_line_1 || "",
        address_line_2: user.profile?.address_line_2 || "",
        city: user.profile?.city || "",
        state: user.profile?.state || "",
        zip_code: user.profile?.zip_code || "",
        phone_number: user.profile?.phone_number || "",
        profile_picture: user.profile?.profile_picture || "",
      },
    },
  });

  return (
    <>
      <Flex>
        <Card className={classes.card} withBorder>
          <form
            className={classes.form}
            onSubmit={(event) => event.preventDefault()}
          >
            <Text fz="xl" fw={700} className={classes.text}>
              Profile Details
            </Text>

            <Divider my={10} />

            <div className={classes.fields}>
              <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
                <TextInput
                  className={classes.field}
                  label="First Name"
                  placeholder="First Name"
                  variant="filled"
                  required
                  {...form.getInputProps("profile.first_name")}
                />
                <TextInput
                  className={classes.field}
                  label="Last Name"
                  placeholder="Last Name"
                  variant="filled"
                  required
                  {...form.getInputProps("profile.last_name")}
                />
              </SimpleGrid>
              <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
                <TextInput
                  className={classes.field}
                  label="Address Line 1"
                  placeholder="Address Line 1"
                  variant="filled"
                  required
                  {...form.getInputProps("profile.address_line_1")}
                />
                <TextInput
                  className={classes.field}
                  label="Address Line 2"
                  placeholder="Address Line 2"
                  variant="filled"
                  {...form.getInputProps("profile.address_line_2")}
                />
              </SimpleGrid>
              <SimpleGrid cols={2} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
                <TextInput
                  className={classes.field}
                  label="City"
                  placeholder="City"
                  variant="filled"
                  required
                  {...form.getInputProps("profile.city")}
                />
                <StateSelect
                  label="State"
                  placeholder="State"
                  variant="filled"
                  required={true}
                  searchable={true}
                  className={classes.field}
                  formProps={form.getInputProps("profile.state")}
                />
              </SimpleGrid>
              <TextInput
                className={classes.field}
                label="Zip Code"
                placeholder="Zip Code"
                variant="filled"
                {...form.getInputProps("profile.zip_code")}
              />
              <Group position="right" mt="md">
                <Button color="white" type="submit" className={classes.control}>
                  Submit
                </Button>
              </Group>
            </div>
          </form>
        </Card>
      </Flex>
    </>
  );
};

export default EditUserProfileDetails;
