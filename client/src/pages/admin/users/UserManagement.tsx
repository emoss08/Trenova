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
import { Text, Card, createStyles, Flex } from "@mantine/core";
import { UsersAdminTable } from "@/components/user-management/table/UserTable";

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
    "&:hover": {
      "& *": {
        color: theme.colors.blue[6],
      },
    },
  },
  grid: {
    display: "flex",
  },
}));

const UserManagement: React.FC = () => {
  const { classes } = useStyles();
  return (
    <>
      <div style={{ flex: 1, marginBottom: 10 }}>
        <Text className={classes.text} fz={20} weight={600}>
          User Management
        </Text>
        <Flex>
          <Text
            color="dimmed"
            size="sm"
            sx={{
              marginRight: "5px",
            }}
          >
            Home -
          </Text>
          <Text
            color="dimmed"
            size="sm"
            sx={{
              marginRight: "5px",
            }}
          >
            User Management -
          </Text>
          <Text color="dimmed" size="sm">
            User List
          </Text>
        </Flex>
      </div>
      <Flex>
        <Card className={classes.card}>
          <UsersAdminTable />
        </Card>
      </Flex>
    </>
  );
};

export default UserManagement;
