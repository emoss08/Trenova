import { parseAsBoolean } from "nuqs";

export const licenseDialog = {
  licenseDialogOpen: parseAsBoolean.withDefault(false),
};

/**
 * Configuration for the license display component
 */
export const licenseConfig = {
  /**
   * List of third-party dependencies and their licenses
   * Add, remove, or modify as needed based on your project dependencies
   */
  thirdPartyLicenses: [
    {
      name: "React",
      license: "MIT License",
      copyright: "Copyright (c) Meta Platforms, Inc. and affiliates.",
      url: "https://github.com/facebook/react/blob/main/LICENSE",
    },
    {
      name: "TypeScript",
      license: "Apache License 2.0",
      copyright: "Copyright (c) Microsoft Corporation.",
      url: "https://github.com/microsoft/TypeScript/blob/main/LICENSE.txt",
    },
    {
      name: "Go",
      license: "BSD 3-Clause License",
      copyright: "Copyright (c) 2009 The Go Authors.",
      url: "https://github.com/golang/go/blob/master/LICENSE",
    },
    {
      name: "PostgreSQL",
      license: "PostgreSQL License",
      copyright: "Copyright (c) The PostgreSQL Global Development Group.",
      url: "https://www.postgresql.org/about/licence/",
    },
    {
      name: "Meilisearch",
      license: "MIT License",
      copyright: "Copyright (c) 2018-present, Meilisearch.",
      url: "https://github.com/meilisearch/meilisearch/blob/main/LICENSE",
    },
    {
      name: "Redis",
      license: "Redis Source Available License v2 (RSALv2)",
      copyright: "Copyright (c) Redis Ltd.",
      url: "https://redis.io/legal/licenses/",
    },
    {
      name: "MinIO",
      license: "GNU AGPL v3.0",
      copyright: "Copyright (c) MinIO, Inc.",
      url: "https://github.com/minio/minio/blob/master/LICENSE",
    },
  ],
};

export default licenseConfig;
