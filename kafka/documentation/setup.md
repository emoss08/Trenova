# Kafka Configuration Guidelines

## Preliminary Setup

### Step 1: Java Installation

Commence the process by installing Java on your system. Execute the following commands:

```shell
sudo apt update
sudo apt install openjdk-11-jdk
```

Check your installation by verifying the Java version:

```shell
java -version
```

### Step 2: Kafka Installation

Install Apache Kafka by downloading it and extracting the compressed file. You may use the following commands:

```shell
wget https://downloads.apache.org/kafka/3.4.0/kafka_2.13-3.4.0.tgz
tar -xzf kafka_2.13-3.4.0.tgz
```

Move the extracted folder to /usr/local/kafka

```shell
sudo mv kafka_2.13-3.4.0 /usr/local/kafka
```

### Step 3: Debezium Connector Installation

Download and extract the Debezium Connector plugin:

```shell
wget https://repo1.maven.org/maven2/io/debezium/debezium-connector-postgres/1.7.0.Final/debezium-connector-postgres-1.7.0.Final-plugin.tar.gz
tar -xzf debezium-connector-postgres-1.7.0.Final-plugin.tar.gz
```

Move the Debezium Connector files to the Kafka library directory:

```shell
sudo mv debezium-connector-postgres /usr/local/kafka/libs
```

Modify or create `connect-standalone.properties` file

```shell
sudo nano /usr/local/kafka/config/connect-standalone.properties
```

Add or replace the following lines to the file

```properties
bootstrap.servers=localhost:9092
# The converters specify the format of data in Kafka and how to translate it into Connect data. 
# Every Connect user will need to configure these based on the format they want their data in when loaded from or stored into Kafka
key.converter=org.apache.kafka.connect.json.JsonConverter
value.converter=org.apache.kafka.connect.json.JsonConverter
key.converter.schemas.enable=true
value.converter.schemas.enable=true
# The internal converter used for Connect offsets. The default is the JSON converter.
internal.key.converter=org.apache.kafka.connect.json.JsonConverter
internal.value.converter=org.apache.kafka.connect.json.JsonConverter
internal.key.converter.schemas.enable=false
internal.value.converter.schemas.enable=false
# Configuration for the Kafka Connect REST API
rest.port=8083
rest.advertised.host.name=127.0.0.1
rest.advertised.port=8083
# The number of tasks that should be created for this connector. The task implementation is responsible for dividing the data up across tasks.
tasks.max=1
# The location of the offset commit data
offset.storage.file.filename=/tmp/connect.offsets
```

Modify or create `debezium-connect-postgres.properties` file

```shell
sudo nano /usr/local/kafka/config/debezium-connect-postgres.properties
```

Add or replace the following lines to the file

```properties
# Set the unique connector name
name=inventory-connector
# Set the connector class to Debezium's PostgreSQL Connector
connector.class=io.debezium.connector.postgresql.PostgresConnector
# Kafka topic to publish data to
topic.prefix=postgres-connector-
# The PostgreSQL server name
database.server.name=localhost
# The PostgreSQL server port
database.port=5432
# The name of the PostgreSQL database
database.dbname=my_database
# The name of the PostgreSQL user
database.user=my_username
# The password for the PostgreSQL user
database.password=my_password
# The maximum number of records that should be loaded into memory while snapshotting
max.batch.size=200
# The maximum number of records per second that should be loaded into memory while snapshotting
max.queue.size=8000
```

Please ensure to adjust the property values according to your database configuration.

### Step 5: PostgreSQL Configuration

Modify the postgresql.conf file to change the Write-Ahead Logging (WAL) level:

```shell
sudo nano /etc/postgresql/12/main/postgresql.conf
```

Add or replace the following lines to the file

```properties
wal_level=logical
```

Ensure to replace '12' with your PostgreSQL version. After editing the file, restart PostgreSQL if it's currently
running:

```shell
sudo systemctl restart postgresql
````

### Step 6: Connection to Kafka Connect REST API

Send a POST request to the Kafka Connect REST API with your configuration details:
```shell
curl -i -X POST -H "Accept:application/json" -H "Content-Type:application/json" localhost:8083/connectors/ -d '{
    "name": "inventory-connector",
    "config": {
        "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
        "database.hostname": "db",
        "database.port": "5432",
        "database.user": "postgres",
        "database.password": "postgres",
        "database.dbname": "postgres",
        "database.server.name": "dbserver1",
        "slot.name": "debezium",
        "plugin.name": "pgoutput",
        "tombstones.on.delete": "false"
    }
}'
```

It's crucial to update the property values to match your specific database configuration.

### Step 7: Verify the Connection

Verify the connection by sending a GET request to the Kafka Connect REST API:

```shell
curl -H "Accept:application/json" localhost:8083/connectors/
```

If the connection was successful, you should see the following response:

```json
["inventory-connector"]
```

You're all set!

Run the following commands to start the Kafka server and Zookeeper:

```shell
sudo /usr/local/kafka/bin/zookeeper-server-start.sh /usr/local/kafka/config/zookeeper.properties

sudo /usr/local/kafka/bin/kafka-server-start.sh /usr/local/kafka/config/server.properties

sudo /usr/local/kafka/bin/connect-standalone.sh /usr/local/kafka/config/connect-standalone.properties /usr/local/kafka/config/debezium-connector-postgres.properties
```
