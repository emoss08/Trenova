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

import {
  Button,
  Col,
  Grid,
  Paper,
  PaperProps,
  Text,
  useMantineTheme,
} from "@mantine/core";
import React from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { IconDefinition } from "@fortawesome/pro-solid-svg-icons";

Alert.defaultProps = {
  withIcon: false,
  icon: undefined,
  withButton: false,
  buttonText: undefined,
  onClick: undefined,
};

type Colors = "blue" | "red" | "yellow" | "green" | "gray" | "dark";

interface Props extends PaperProps {
  title: React.ReactNode;
  message: string;
  withIcon?: boolean;
  icon?: IconDefinition;
  withButton?: boolean;
  buttonText?: React.ReactNode;
  onClick?: () => void;
  color: Colors;
}

export function Alert({
  title,
  message,
  withIcon,
  icon,
  withButton,
  buttonText,
  onClick,
  color,
  ...rest
}: Props) {
  const theme = useMantineTheme();
  const borderColorValue: Record<string, string> = {
    blue: "#007BFF",
    red: "#ff0000",
    yellow: "#ffc107",
    green: "#28a745",
    gray: "#6c757d",
    dark: "#343a40",
  };

  const backgroundColorValue: Record<string, string> = {
    blue:
      theme.colorScheme === "dark" ? "rgb(33, 46, 72)" : "rgb(241, 250, 255)",
    red:
      theme.colorScheme === "dark" ? "rgb(72, 33, 33)" : "rgb(255, 241, 241)",
    yellow:
      theme.colorScheme === "dark" ? "rgb(72, 72, 33)" : "rgb(255, 255, 241)",
    green:
      theme.colorScheme === "dark" ? "rgb(33, 72, 33)" : "rgb(241, 255, 241)",
    gray:
      theme.colorScheme === "dark" ? "rgb(72, 72, 72)" : "rgb(241, 241, 241)",
    dark:
      theme.colorScheme === "dark" ? "rgb(33, 33, 33)" : "rgb(241, 241, 241)",
  };

  const borderColorChoice = borderColorValue[color];
  const backgroundColorChoice = backgroundColorValue[color];
  const iconColorChoice = borderColorValue[color];
  const shouldRenderButton = withButton === undefined ? false : withButton;
  const shouldRenderIcon = withIcon === undefined ? false : withIcon;

  return (
    <Paper
      p="lg"
      shadow="xs"
      style={{
        borderColor: borderColorChoice,
        borderRadius: "6.175px",
        textAlign: "start",
        borderStyle: "dashed",
        backgroundColor: backgroundColorChoice,
        borderWidth: "1px",
      }}
      {...rest}
    >
      <Grid>
        <Col
          style={{
            display: "flex",
            flexDirection: "row",
            alignItems: "center",
          }}
        >
          {shouldRenderIcon && icon && (
            <FontAwesomeIcon
              icon={icon}
              style={{
                marginRight: 15,
                marginBottom: 30,
                color: iconColorChoice,
              }}
              size="2xl"
            />
          )}
          <div style={{ flexGrow: 1 }}>
            <Text fw={700}>{title}</Text>
            <Text fw={400} c="dimmed" style={{ paddingTop: 5 }}>
              {message}
            </Text>
          </div>
          {shouldRenderButton && (
            <Button
              color={color}
              size="md"
              style={{ marginLeft: 10 }}
              radius={5}
              onClick={onClick}
            >
              {buttonText}
            </Button>
          )}
        </Col>
      </Grid>
    </Paper>
  );
}
