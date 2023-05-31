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

import { JobTitle, User } from "@/types/user";
import React from "react";
import {
  Avatar,
  Card,
  Container,
  createStyles,
  Flex,
  Grid,
  SimpleGrid,
  Text,
} from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faCircleUser,
  faEnvelope,
  faMapPin,
} from "@fortawesome/pro-duotone-svg-icons";

type Props = {
  user: User;
  jobTitle: JobTitle;
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

const UserProfileDetails: React.FC<Props> = ({ user, jobTitle }) => {
  const { classes } = useStyles();

  return (
    <>
      <Flex>
        <Card className={classes.card} withBorder>
          <Container mx="xs" my="xs">
            <SimpleGrid cols={2} className={classes.grid}>
              <Avatar src={user.profile?.profile_picture} size={200} />
              <Grid className={classes.grid}>
                <Grid.Col>
                  <Flex direction="column" justify="start">
                    <Text className={classes.text} fz="35px" fw={650}>
                      {user.profile?.first_name} {user.profile?.last_name}
                    </Text>
                    <Grid grow gutter={30} align="flex-start">
                      <Grid.Col span={1}>
                        <div className={classes.div}>
                          <FontAwesomeIcon
                            icon={faCircleUser}
                            className={classes.icon}
                          />
                          <Text className={classes.text}>{jobTitle.name}</Text>
                        </div>
                      </Grid.Col>
                      <Grid.Col span={6}>
                        <div className={classes.div}>
                          <FontAwesomeIcon
                            icon={faMapPin}
                            className={classes.icon}
                          />
                          <Text className={classes.text}>
                            {user.profile?.address_line_1} {user.profile?.city}{" "}
                            {user.profile?.state} {user.profile?.zip_code}
                          </Text>
                        </div>
                      </Grid.Col>
                      <Grid.Col span={1}>
                        <div className={classes.div}>
                          <FontAwesomeIcon
                            icon={faEnvelope}
                            className={classes.icon}
                          />
                          <Text className={classes.text}>{user.email}</Text>
                        </div>
                      </Grid.Col>
                    </Grid>
                    <Text mt={20}>
                      TODO: Add Analytics based on job function
                    </Text>
                  </Flex>
                </Grid.Col>
              </Grid>
            </SimpleGrid>
          </Container>
        </Card>
      </Flex>
    </>
  );
};

export default UserProfileDetails;
