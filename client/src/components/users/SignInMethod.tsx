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
import { Card, createStyles, Divider, Flex, Text } from "@mantine/core";
import { faShieldCheck } from "@fortawesome/pro-duotone-svg-icons";
import { Alert } from "@/components/ui/Alert";
import { EmailChangeForm } from "@/components/users/EmailChange";
import { PasswordChangeForm } from "@/components/users/PasswordChange";

type Props = {
  user: User;
};

const useStyles = createStyles((theme) => ({
  card: {
    width: "100%",
    maxWidth: "100%",
    height: "auto",
    "@media (max-width: 576px)": {
      height: "auto",
      maxHeight: "none",
    },
  },
  text: {
    color: theme.colorScheme === "dark" ? "white" : "black",
  },
  icon: {
    marginRight: "5px",
    marginTop: "5px",
  },
  div: {
    display: "flex",
  },
  grid: {
    display: "flex",
  },
}));

export const SignInMethod: React.FC<Props> = ({ user }) => {
  const { classes } = useStyles();

  return (
    <>
      <Flex>
        <Card className={classes.card} withBorder>
          <Text fz="xl" fw={700} className={classes.text}>
            Sign-In Method
          </Text>

          <Divider my={10} />

          <EmailChangeForm user={user} />
          <PasswordChangeForm />
          <Alert
            color="blue"
            icon={faShieldCheck}
            title="Secure your account"
            message={`Two-factor authentication adds an extra layer of security to your account. 
            To log in, in addition you'll need to provide a 6 digit code`}
            buttonText="Enable"
            withButton
            withIcon
          />
        </Card>
      </Flex>
    </>
  );
};
