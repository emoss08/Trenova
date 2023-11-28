#!/bin/bash

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
  echo "Please enter the database password for the new PostgreSQL user 'monta_user':"
  read -s DB_PASSWORD

  sudo apt-get update
  sudo apt-get install -y postgresql postgresql-contrib

  # Configure PostgreSQL
  sudo -u postgres psql -c "CREATE DATABASE monta;"
  sudo -u postgres psql -c "CREATE USER monta_user WITH PASSWORD '$DB_PASSWORD';"
  sudo -u postgres psql -c "ALTER ROLE monta_user SET client_encoding TO 'utf8';"
  sudo -u postgres psql -c "ALTER ROLE monta_user SET default_transaction_isolation TO 'read committed';"
  sudo -u postgres psql -c "ALTER ROLE monta_user SET timezone TO 'UTC';"
  sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE monta TO monta_user;"
EOF

# Clone the Monta project from Git
ssh -i $SSH_PRIVATE_KEY_FILE $SSH_USER@"$INSTANCE_IP" <<EOF
  git clone $GIT_REPO_URL
EOF

# Install the requirements for the Monta app
ssh -i $SSH_PRIVATE_KEY_FILE $SSH_USER@$INSTANCE_IP <<EOF
  cd monta
  pip3 install -r requirements.txt

  # Frontend setup
  npm install

  # Django setup
  python3 manage.py migrate
  py manage.py createsystemuser --username monta --password monta --organization "Monta Transportation"
EOF