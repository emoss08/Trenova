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

import { Divider, Modal, ScrollArea, Table, Text } from "@mantine/core";
import React from "react";
import { billingClientStore } from "@/stores/BillingStores";
import { WebsocketMessageProps } from "@/utils/websockets";

interface Props {
  websocketMessage: WebsocketMessageProps;
}

export const BillingExceptionModal: React.FC<Props> = ({
  websocketMessage,
}) => {
  const [modalOpen, setModalOpen] =
    billingClientStore.use("exceptionModalOpen");

  if (!modalOpen) return null;

  console.log("Websocket message in ExceptionModal", websocketMessage);

  return (
    <Modal.Root
      opened={modalOpen}
      onClose={() => setModalOpen(false)}
      centered
      styles={{
        inner: {
          section: {
            overflowY: "visible",
          },
        },
      }}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>
            Billing Exceptions
            <Text size="xs" color="dimmed" mt={2}>
              The following invoices have failed due to missing documentation.
            </Text>
          </Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Divider mb={5} />
          <ScrollArea h="auto">
            <Table striped highlightOnHover>
              <thead>
                <tr>
                  <th>Invoice Number</th>
                  <th>Missing Documents</th>
                </tr>
              </thead>
              <tbody>
                {typeof websocketMessage.payload.message !== "string" ? (
                  websocketMessage.payload.message?.flatMap((subArray, index) =>
                    subArray.map((item, subIndex) => (
                      <tr key={`${index}-${subIndex}`}>
                        <td>{item.invoice_number}</td>
                        <td>{item.missing_documents.join(", ")}</td>
                      </tr>
                    ))
                  )
                ) : (
                  <tr>
                    <td colSpan={2}>{websocketMessage.payload.message}</td>
                  </tr>
                )}
              </tbody>
            </Table>
          </ScrollArea>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
};
