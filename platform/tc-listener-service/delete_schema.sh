#!/bin/bash
# Copyright (c) 2024 Trenova Technologies, LLC
#
# Licensed under the Business Source License 1.1 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://trenova.app/pricing/
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#
# Key Terms:
# - Non-production use only
# - Change Date: 2026-11-16
# - Change License: GNU General Public License v2 or later
#
# For full license text, see the LICENSE file in the root directory.


# Set the URL for your Schema Registry
SCHEMA_REGISTRY_URL="http://localhost:8081"

# Fetch all subjects
SUBJECTS=$(curl -s -X GET "$SCHEMA_REGISTRY_URL/subjects")

# Extract subjects from the JSON response
SUBJECT_LIST=$(echo $SUBJECTS | grep -o '"[^"]*"' | sed 's/"//g')

# Loop through each subject and delete it
for SUBJECT in $SUBJECT_LIST; do
  echo "Deleting subject $SUBJECT"
  curl -X DELETE "$SCHEMA_REGISTRY_URL/subjects/$SUBJECT"
done

echo "All subjects have been deleted."
