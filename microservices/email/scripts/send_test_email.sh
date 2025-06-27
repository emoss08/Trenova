#!/bin/bash

# Change to the project root directory
cd "$(dirname "$0")/.."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in the PATH"
    exit 1
fi

# Set default values
to_email=""
template="welcome"
subject="Test Email from Trenova"
rabbitmq_host="localhost"
rabbitmq_port=5677
rabbitmq_user="guest"
rabbitmq_pass="guest"
rabbitmq_exchange="trenova.events"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --to)
      to_email="$2"
      shift 2
      ;;
    --template)
      template="$2"
      shift 2
      ;;
    --subject)
      subject="$2"
      shift 2
      ;;
    --host)
      rabbitmq_host="$2"
      shift 2
      ;;
    --port)
      rabbitmq_port="$2"
      shift 2
      ;;
    --user)
      rabbitmq_user="$2"
      shift 2
      ;;
    --pass)
      rabbitmq_pass="$2"
      shift 2
      ;;
    --exchange)
      rabbitmq_exchange="$2"
      shift 2
      ;;
    *)
      echo "Unknown parameter: $1"
      exit 1
      ;;
  esac
done

# Validate required parameters
if [ -z "$to_email" ]; then
    echo "Error: Recipient email (--to) is required"
    echo "Usage: $0 --to email@example.com [--template welcome] [--subject \"Test Email\"] [--host localhost] [--port 5673] [--user guest] [--pass guest] [--exchange trenova.events]"
    exit 1
fi

# Check if the service is running in Docker Compose
if docker-compose ps | grep -q "trenova-email-service" && docker-compose ps | grep -q "Up"; then
    echo "Email service is running in Docker Compose"
    echo "Using Docker Compose to run the test script..."
    
    # Run the script in the container
    docker-compose exec email-service go run scripts/send_test_email.go \
        -to="$to_email" \
        -template="$template" \
        -subject="$subject" \
        -host="$rabbitmq_host" \
        -port="$rabbitmq_port" \
        -user="$rabbitmq_user" \
        -pass="$rabbitmq_pass" \
        -exchange="$rabbitmq_exchange"
else
    echo "Running test script locally..."
    
    # Run the script directly
    go run scripts/send_test_email.go \
        -to="$to_email" \
        -template="$template" \
        -subject="$subject" \
        -host="$rabbitmq_host" \
        -port="$rabbitmq_port" \
        -user="$rabbitmq_user" \
        -pass="$rabbitmq_pass" \
        -exchange="$rabbitmq_exchange"
fi

echo "Done!" 