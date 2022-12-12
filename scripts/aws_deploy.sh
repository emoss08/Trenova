#!/usr/bin/env bash
: '
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.

-------------------------------------------------------------------------------

This script is used to deploy the django app to AWS.
'

# Set the AWS access key and secret key
export AWS_ACCESS_KEY_ID="YOUR_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="YOUR_SECRET_KEY"

# Set the region
export AWS_REGION="YOUR_REGION"

# Set the AMI to use for the EC2 instance
AMI="YOUR_AMI"

# Set the instance type
INSTANCE_TYPE="YOUR_INSTANCE_TYPE"

# Set the SSH user
SSH_USER="YOUR_SSH_USER"

# Set the path to the SSH private key file
SSH_PRIVATE_KEY_FILE="YOUR_SSH_PRIVATE_KEY_FILE"

# Set the URL of the Git repository for the Monta project
GIT_REPO_URL="git@github.com:Monta-Application/Monta.git"

# Create the EC2 instance
aws ec2 run-instances \
  --image-id $AMI \
  --instance-type $INSTANCE_TYPE \
  --security-group-ids "$SECURITY_GROUP_ID" \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=Monta app}]'

# Wait for the instance to be created
aws ec2 wait instance-running

# Get the public IP address of the EC2 instance
INSTANCE_IP=$(aws ec2 describe-instances \
  --filters "Name=tag:Name,Values=Monta app" \
  --query "Reservations[*].Instances[*].PublicIpAddress" \
  --output text)

# SSH into the EC2 instance and install the required packages
ssh -i $SSH_PRIVATE_KEY_FILE $SSH_USER@"$INSTANCE_IP" <<EOF
  sudo apt-get update
  sudo apt-get install -y python3-pip python3-dev libpq-dev postgresql postgresql-contrib
EOF

# Clone the Monta project from Git
# shellcheck disable=SC2087
ssh -i $SSH_PRIVATE_KEY_FILE $SSH_USER@"$INSTANCE_IP" <<EOF
  git clone $GIT_REPO_URL
EOF

# Install the requirements for the Monta app
# shellcheck disable=SC2086
ssh -i $SSH_PRIVATE_KEY_FILE $SSH_USER@$INSTANCE_IP <<EOF
  cd monta
  pip3 install -r requirements.txt
EOF
