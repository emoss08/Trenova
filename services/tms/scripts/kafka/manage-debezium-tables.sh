#!/bin/bash

CONNECTOR_NAME="trenova-postgres-connector"
CONNECT_URL="http://localhost:8083"

show_help() {
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  status                    Show connector status"
    echo "  list-tables              List all monitored tables"
    echo "  add-table <table>        Add a table to monitoring"
    echo "  remove-table <table>     Remove a table from monitoring"
    echo "  set-tables <table1,table2,...>  Set exact list of tables"
    echo "  enable-all               Monitor all tables in public schema"
    echo "  restart                  Restart the connector"
    echo "  delete                   Delete the connector"
    echo ""
    echo "Examples:"
    echo "  $0 status"
    echo "  $0 add-table public.new_table"
    echo "  $0 set-tables public.shipments,public.users"
    echo "  $0 enable-all"
}

get_connector_config() {
    curl -s "$CONNECT_URL/connectors/$CONNECTOR_NAME/config" | jq -r .
}

get_table_list() {
    local config=$(get_connector_config)
    if echo "$config" | jq -e '.["table.include.list"]' > /dev/null 2>&1; then
        echo "$config" | jq -r '.["table.include.list"]'
    elif echo "$config" | jq -e '.["schema.include.list"]' > /dev/null 2>&1; then
        echo "ALL tables in schema: $(echo "$config" | jq -r '.["schema.include.list"]')"
    else
        echo "No table configuration found"
    fi
}

update_connector_config() {
    local new_config="$1"
    curl -X PUT "$CONNECT_URL/connectors/$CONNECTOR_NAME/config" \
        -H "Content-Type: application/json" \
        -d "$new_config"
}

case "$1" in
    "status")
        echo "Connector Status:"
        curl -s "$CONNECT_URL/connectors/$CONNECTOR_NAME/status" | jq .
        ;;
    
    "list-tables")
        echo "Currently monitored tables:"
        get_table_list
        ;;
    
    "add-table")
        if [ -z "$2" ]; then
            echo "Error: Please specify a table name"
            echo "Usage: $0 add-table public.table_name"
            exit 1
        fi
        
        current_tables=$(get_table_list)
        if [[ "$current_tables" == *"ALL tables"* ]]; then
            echo "Currently monitoring ALL tables. Use 'set-tables' to switch to specific table list."
            exit 1
        fi
        
        new_tables="$current_tables,$2"
        new_config=$(get_connector_config | jq --arg tables "$new_tables" '. + {"table.include.list": $tables} | del(.["schema.include.list"])')
        
        echo "Adding table: $2"
        update_connector_config "$new_config"
        ;;
    
    "remove-table")
        if [ -z "$2" ]; then
            echo "Error: Please specify a table name"
            exit 1
        fi
        
        current_tables=$(get_table_list)
        if [[ "$current_tables" == *"ALL tables"* ]]; then
            echo "Currently monitoring ALL tables. Use 'set-tables' to switch to specific table list."
            exit 1
        fi
        
        new_tables=$(echo "$current_tables" | sed "s/$2,//g" | sed "s/,$2//g" | sed "s/^$2$//g")
        new_config=$(get_connector_config | jq --arg tables "$new_tables" '. + {"table.include.list": $tables}')
        
        echo "Removing table: $2"
        update_connector_config "$new_config"
        ;;
    
    "set-tables")
        if [ -z "$2" ]; then
            echo "Error: Please specify table list"
            echo "Usage: $0 set-tables public.table1,public.table2"
            exit 1
        fi
        
        new_config=$(get_connector_config | jq --arg tables "$2" '. + {"table.include.list": $tables} | del(.["schema.include.list"])')
        
        echo "Setting tables to: $2"
        update_connector_config "$new_config"
        ;;
    
    "enable-all")
        new_config=$(get_connector_config | jq '. + {"schema.include.list": "public"} | del(.["table.include.list"])')
        
        echo "Enabling monitoring for ALL tables in public schema"
        update_connector_config "$new_config"
        ;;
    
    "restart")
        echo "Restarting connector..."
        curl -X POST "$CONNECT_URL/connectors/$CONNECTOR_NAME/restart"
        ;;
    
    "delete")
        echo "Deleting connector..."
        curl -X DELETE "$CONNECT_URL/connectors/$CONNECTOR_NAME"
        ;;
    
    *)
        show_help
        ;;
esac