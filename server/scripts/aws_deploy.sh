#!/bin/bash

#
# COPYRIGHT(c) 2024 Trenova
#
# This file is part of Trenova.
#
# The Trenova software is licensed under the Business Source License 1.1. You are granted the right
# to copy, modify, and redistribute the software, but only for non-production use or with a total
# of less than three server instances. Starting from the Change Date (November 16, 2026), the
# software will be made available under version 2 or later of the GNU General Public License.
# If you use the software in violation of this license, your rights under the license will be
# terminated automatically. The software is provided "as is," and the Licensor disclaims all
# warranties and conditions. If you use this license's text or the "Business Source License" name
# and trademark, you must comply with the Licensor's covenants, which include specifying the
# Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
# Grant, and not modifying the license in any other way.
#


# Load sensitive data from environment variables
AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
AWS_REGION=$AWS_REGION
AMI=$AMI
INSTANCE_TYPE=$INSTANCE_TYPE
SSH_USER=$SSH_USER
SSH_PRIVATE_KEY_FILE=$SSH_PRIVATE_KEY_FILE
GIT_REPO_URL=$GIT_REPO_URL
SECURITY_GROUP_ID=$SECURITY_GROUP_ID

# Create the EC2 instance
aws ec2 run-instances \
  --image-id $AMI \
  --instance-type $INSTANCE_TYPE \
  --security-group-ids "$SECURITY_GROUP_ID" \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=Trenova app}]'

# Wait for the instance to be created
aws ec2 wait instance-running

# Get the public IP address of the EC2 instance
INSTANCE_IP=$(aws ec2 describe-instances \
  --filters "Name=tag:Name,Values=Trenova app" \
  --query "Reservations[*].Instances[*].PublicIpAddress" \
  --output text)

# SSH into the EC2 instance and install the required packages
ssh -i $SSH_PRIVATE_KEY_FILE $SSH_USER@"$INSTANCE_IP" <<EOF
  sudo apt-get update
  sudo apt-get install -y python3-pip python3-dev libpq-dev git

  # Additional dependencies for frontend
  curl -sL https://deb.nodesource.com/setup_14.x | sudo -E bash -
  sudo apt-get install -y nodejs

  # Install Kafka, Zookeeper, Debezium
  # [Include installation steps for Kafka, Zookeeper, Debezium here]

  # Install Redis
  sudo apt-get install -y redis-server
EOF
# SSH into the EC2 instance for PostgreSQL setup
ssh -i $SSH_PRIVATE_KEY_FILE $SSH_USER@"$INSTANCE_IP" <<'EOF'
  echo "Please enter the database password for the new PostgreSQL user 'Trenova_user':"
  read -s DB_PASSWORD

  sudo apt-get update
  sudo apt-get install -y postgresql postgresql-contrib

  # Configure PostgreSQL
  sudo -u postgres psql -c "CREATE DATABASE Trenova;"
  sudo -u postgres psql -c "CREATE USER Trenova_user WITH PASSWORD '$DB_PASSWORD';"
  sudo -u postgres psql -c "ALTER ROLE Trenova_user SET client_encoding TO 'utf8';"
  sudo -u postgres psql -c "ALTER ROLE Trenova_user SET default_transaction_isolation TO 'read committed';"
  sudo -u postgres psql -c "ALTER ROLE Trenova_user SET timezone TO 'UTC';"
  sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE Trenova TO Trenova_user;"
EOF

# Clone the Trenova project from Git
ssh -i $SSH_PRIVATE_KEY_FILE $SSH_USER@"$INSTANCE_IP" <<EOF
  git clone $GIT_REPO_URL
EOF

# Install the requirements for the Trenova app
ssh -i $SSH_PRIVATE_KEY_FILE $SSH_USER@$INSTANCE_IP <<EOF
  cd Trenova
  pip3 install -r requirements.txt

  # Frontend setup
  npm install

  # Django setup
  python3 manage.py migrate
  py manage.py createsystemuser --username Trenova --password Trenova --organization "Trenova Transportation"
EOF