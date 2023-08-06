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

import React, { useState } from "react";
import {
  Modal,
  TextInput,
  Text,
  Group,
  Badge,
  ScrollArea,
  rem,
  Anchor,
  Divider,
  Table,
  Pagination,
  Box,
} from "@mantine/core";
import { useNavigate } from "react-router-dom";
import { useDisclosure } from "@mantine/hooks";
import { useDebouncedCallback } from "use-debounce";
import { faSearch } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { SearchControl } from "@/components/layout/Header/_Partials/SpotlightSearchControl";
import axios from "@/lib/AxiosConfig";
import { allowedSearchModels } from "@/utils/apps/search";

export const SearchModal: React.FC = () => {
  const navigate = useNavigate();
  const [opened, { open, close }] = useDisclosure(false);
  const [input, setInput] = useState("");
  const [results, setResults] = useState<any[]>([]);
  const [badge, setBadge] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [, setNextPage] = useState<string | null>(null);
  const [, setPreviousPage] = useState<string | null>(null);
  const [count, setCount] = useState(0);

  // Update search results when page changes
  React.useEffect(() => {
    handleSearch(input, page);
  }, [page]);

  // Update badge when input changes
  React.useEffect(() => {
    const [potentialModel, _] = input.split(":").map((s) => s.trim());
    if (allowedSearchModels.includes(potentialModel)) {
      setBadge(potentialModel);
    } else {
      setBadge(null);
    }
  }, [input]);

  // Debounced search handler
  const handleSearch = useDebouncedCallback(
    async (input: string, page: number) => {
      try {
        const response = await axios.get(
          `search/?term=${encodeURIComponent(input)}&page=${page}`
        );
        setCount(response.data.pages);
        setResults(response.data.results);

        if (response.data.next) {
          setNextPage(response.data.next);
        } else {
          setNextPage(null);
        }

        if (response.data.previous) {
          setPreviousPage(response.data.previous);
        } else {
          setPreviousPage(null);
        }
      } catch (error) {
        console.error("Error fetching search results:", error);
      }
    },
    500
  ); // Debounce time in ms

  return (
    <>
      <SearchControl onClick={open} />
      <Modal.Root opened={opened} onClose={close} size="xl">
        <Modal.Overlay />
        <Modal.Content>
          <Modal.Header px={0} py={0}>
            <Modal.Title style={{ width: "100%" }}>
              <TextInput
                placeholder="Search..."
                variant="unstyled"
                value={input}
                icon={<FontAwesomeIcon icon={faSearch} />}
                size="lg"
                onChange={(event) => {
                  setInput(event.currentTarget.value);
                  handleSearch(event.currentTarget.value, page);
                  // Reset page to 1
                  setPage(1);
                  // Reset count
                  setCount(0);
                  // Set results to empty
                  setResults([]);
                }}
                styles={{
                  rightSection: {
                    pointerEvents: "none",
                    // make sure that is full size
                    width: "auto",
                    // add some padding to the right
                    paddingRight: 10,
                  },
                }}
                rightSection={
                  badge && (
                    <Badge
                      variant="dot"
                      color="violet"
                      radius="sm"
                      style={{
                        // background color light red
                        backgroundColor: "#e0ccff",
                        // border color dark red
                        borderColor: "#9552fa",
                        // Text color dark red
                        color: "#9552fa",
                      }}
                    >
                      {badge}
                    </Badge>
                  )
                }
              />
            </Modal.Title>
            {/* <Modal.CloseButton /> */}
          </Modal.Header>
          <Divider />

          <Modal.Body>
            <ScrollArea offsetScrollbars h={400} mt={10} scrollbarSize={5}>
              {results.length > 0 ? (
                <Table striped highlightOnHover>
                  <thead>
                    <tr>
                      <th>App Name</th>
                      <th>Results</th>
                    </tr>
                  </thead>
                  <tbody>
                    {results.map((result, index) => (
                      <tr
                        key={index}
                        style={{
                          marginBottom: "10px",
                          cursor: "pointer",
                        }}
                        onClick={() => {
                          console.log("clicked", result.display);
                          // navigate(result.path);
                        }}
                      >
                        <td>{result.model_name}</td>
                        <td>{result.display}</td>
                      </tr>
                    ))}
                  </tbody>
                </Table>
              ) : (
                <Text size="lg" align="center" style={{ marginTop: "20px" }}>
                  No Results
                </Text>
              )}
            </ScrollArea>
          </Modal.Body>
          {/* Center div with pagination */}
          <Box
            style={{
              justifyContent: "center",
              display: "flex",
            }}
            pb={10}
          >
            {results.length > 0 && (
              <Box
                style={{
                  justifyContent: "center",
                  display: "flex",
                }}
                pb={10}
              >
                <Pagination value={page} onChange={setPage} total={count} />
              </Box>
            )}
          </Box>
          <div>
            <Group
              position="apart"
              px={15}
              py="xs"
              sx={(theme) => ({
                borderTop: `${rem(1)} solid ${
                  theme.colorScheme === "dark"
                    ? theme.colors.dark[4]
                    : theme.colors.gray[2]
                }`,
              })}
            >
              <Text size="xs" color="dimmed">
                Search powered by Monta
              </Text>
              <Anchor size="xs" href="#">
                Learn more
              </Anchor>
            </Group>
          </div>
        </Modal.Content>
      </Modal.Root>
    </>
  );
};
