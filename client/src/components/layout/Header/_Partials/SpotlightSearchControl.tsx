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
import { faMagnifyingGlass } from "@fortawesome/pro-solid-svg-icons";
import { spotlight } from "@mantine/spotlight";
import { createStyles, Group, UnstyledButton } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

const useStyles = createStyles((theme) => ({
  button: {
    backgroundColor:
      theme.colorScheme === "dark"
        ? "rgb(37, 38, 43)"
        : "rgba(248, 249, 250, 0.35)",
    borderRadius: "8px",
    borderStyle: "solid",
    borderColor:
      theme.colorScheme === "dark" ? "rgb(55, 58, 64)" : "rgb(222, 226, 230)",
    borderWidth: "1px",
    color:
      theme.colorScheme === "dark"
        ? "rgb(144, 146, 150)"
        : "rgb(173, 181, 189)",
    fontSize: "16px",
    paddingLeft: "12px",
    paddingRight: "5px",
    textAlign: "left",
    height: "34px",
    ":hover": {
      backgroundColor:
        theme.colorScheme === "dark"
          ? "rgb(39, 39, 43)"
          : "rgba(249, 250, 250, 0.35)",
    },
    "&:hover svg": {
      color:
        theme.colorScheme === "dark"
          ? theme.colors.gray[0]
          : theme.colors.gray[6],
    },
  },
  text: {
    backgroundColor:
      theme.colorScheme === "dark" ? "#2c2d33" : "rgb(248, 249, 250)",
    borderRadius: "4px",
    borderWidth: "1px",
    borderStyle: "solid",
    borderColor:
      theme.colorScheme === "dark" ? "#2c2d33" : "rgb(233, 236, 239)",
    boxSizing: "border-box",
    color: theme.colorScheme === "dark" ? "#c1c2c5" : "rgb(73, 80, 87)",
    fontSize: "11px",
    fontWeight: 700,
    paddingLeft: "7px",
    paddingRight: "7px",
    textAlign: "left",
    marginLeft: "1px",
  },
  mainText: {
    textAlign: "left",
    fontSize: "14px",
    color: "rgb(144, 146, 150)",
    paddingRight: "40px",
    textDecorationStyle: "solid",
  },
  group: {
    alignItems: "center",
    color: theme.colorScheme === "dark" ? "#c1c2c5" : "#c1c2c5",
    columnGap: "10px",
    display: "flex",
    flexDirection: "row",
    flexWrap: "wrap",
    fontSize: "16px",
    justifyContent: "flex-start",
    lineHeight: "18.4px",
    rowGap: "10px",
    textAlign: "left",
  },
}));

export const SearchControl = () => {
  const { classes } = useStyles();

  return (
    <UnstyledButton onClick={() => spotlight.open()} className={classes.button}>
      <Group className={classes.group}>
        <FontAwesomeIcon size={"xs"} icon={faMagnifyingGlass} />
        <div className={classes.mainText}>Search</div>
        <div className={classes.text}>Ctrl + K</div>
      </Group>
    </UnstyledButton>
  );
};
