#!/bin/bash

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
