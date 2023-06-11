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

import React, { ChangeEvent, useState } from "react";
import { Avatar, Input, Button, ActionIcon, Tooltip } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faPencil } from "@fortawesome/pro-solid-svg-icons";
import { faXmark } from "@fortawesome/pro-regular-svg-icons";
import { User } from "@/types/user";
import axios from "@/lib/axiosConfig";
import { getUserAuthToken } from "@/lib/utils";
import { notifications } from "@mantine/notifications";
import { useQueryClient } from "react-query";

interface AvatarInputProps {
  defaultAvatar?: string;
  user: User;
}

const AvatarInput: React.FC<AvatarInputProps> = ({ defaultAvatar, user }) => {
  const [avatar, setAvatar] = useState<string | null>(defaultAvatar ?? null);
  const [showRemove, setShowRemove] = useState<boolean>(false);
  const queryClient = useQueryClient();

  const handleFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    if (event.target.files && event.target.files[0]) {
      const imgUrl = URL.createObjectURL(event.target.files[0]);
      setAvatar(imgUrl);
      setShowRemove(true);

      const formData = new FormData();
      formData.append("profile.profile_picture", event.target.files[0]);

      try {
        await axios.patch(`/users/${user.id}/`, formData, {
          headers: {
            "Content-Type": "multipart/form-data",
            Authorization: `Bearer ${getUserAuthToken()}`,
          },
        });
      } catch (error) {
        console.error(error);
      } finally {
        queryClient.invalidateQueries("user").then(() => {
          notifications.show({
            title: "Success",
            message: "Your profile picture has been updated.",
            color: "green",
            withCloseButton: true,
            icon: <FontAwesomeIcon icon={faCheck} />,
          });
        });
      }
    }
  };

  const removeAvatar = async () => {
    setAvatar(null);
    setShowRemove(false);

    const formData = new FormData();
    formData.append("profile.profile_picture", ""); // empty string will remove the picture

    try {
      const response = await axios.patch(`/users/${user.id}/`, formData, {
        headers: {
          "Content-Type": "multipart/form-data",
          Authorization: `Bearer ${getUserAuthToken()}`,
        },
      });
      console.log(response.data);
    } catch (error) {
      console.error(error);
    } finally {
      queryClient.invalidateQueries("user").then(() => {
        notifications.show({
          title: "Success",
          message: "Your profile picture has been removed.",
          color: "green",
          withCloseButton: true,
          icon: <FontAwesomeIcon icon={faCheck} />,
        });
      });
    }
  };
  return (
    <>
      <div style={{ position: "relative", width: 230, height: 200 }}>
        {avatar ? (
          <Avatar src={avatar} size={200} />
        ) : (
          <Avatar color="cyan" size={200}>
            {user.profile?.first_name.charAt(0)}
            {user.profile?.last_name.charAt(0)}
          </Avatar>
        )}

        {avatar && (
          <label>
            <Tooltip withArrow label="Remove Avatar">
              <ActionIcon
                component="span"
                radius="lg"
                style={{ position: "absolute", right: 20, bottom: -5 }}
                variant="filled"
                size="sm"
                onClick={removeAvatar}
              >
                <FontAwesomeIcon icon={faXmark} size="xs" />
              </ActionIcon>
            </Tooltip>
          </label>
        )}
        <label>
          <Input
            type="file"
            accept=".png, .jpg, .jpeg"
            onChange={handleFileChange}
            style={{ display: "none" }}
          />
          <Tooltip withArrow label={avatar ? "Change Avatar" : "Add Avatar"}>
            <ActionIcon
              component="span"
              radius="lg"
              style={{ position: "absolute", right: 20, top: -5 }}
              variant="filled"
              size="sm"
            >
              <FontAwesomeIcon icon={faPencil} size="2xs" />
            </ActionIcon>
          </Tooltip>
        </label>
      </div>
    </>
  );
};

export default AvatarInput;