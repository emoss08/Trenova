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
import { Card, Container, Flex, Grid, SimpleGrid, Text } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faCircleUser,
  faEnvelope,
  faMapPin,
} from "@fortawesome/pro-duotone-svg-icons";
import { JobTitle, User } from "@/types/apps/accounts";
import { AvatarInput } from "../ui/fields/AvatarInput";
import { usePageStyles } from "@/styles/PageStyles";

type ViewUserProfileDetailsProps = {
  user: User;
  jobTitle: JobTitle;
};

export function ViewUserProfileDetails({
  user,
  jobTitle,
}: ViewUserProfileDetailsProps) {
  const { classes } = usePageStyles();

  return (
    <Flex>
      <Card className={classes.card}>
        <Container mx="xs" my="xs">
          <SimpleGrid cols={3} className={classes.grid}>
            <AvatarInput
              defaultAvatar={user.profile?.profilePicture}
              user={user}
            />
            <Grid grow className={classes.grid}>
              <Grid.Col>
                <Flex direction="column" justify="start">
                  <Text className={classes.text} fz="35px" fw={650}>
                    {user.profile?.firstName} {user.profile?.lastName}
                  </Text>
                  <Grid grow gutter={30} align="flex-start">
                    <Grid.Col span={1}>
                      <div className={classes.div}>
                        <FontAwesomeIcon
                          icon={faCircleUser}
                          className={classes.icon}
                        />
                        <Text className={classes.text}>
                          {jobTitle.name ?? ""}
                        </Text>
                      </div>
                    </Grid.Col>
                    <Grid.Col span={5}>
                      <div className={classes.div}>
                        <FontAwesomeIcon
                          icon={faMapPin}
                          className={classes.icon}
                        />
                        <Text className={classes.text}>
                          {user.profile?.addressLine1}{" "}
                          {user.profile?.addressLine2 ?? ""}{" "}
                          {user.profile?.city} {user.profile?.state}{" "}
                          {user.profile?.zipCode}
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
                  <Text mt={20}>TODO: Add Analytics based on job function</Text>
                </Flex>
              </Grid.Col>
            </Grid>
          </SimpleGrid>
        </Container>
      </Card>
    </Flex>
  );
}
