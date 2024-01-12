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

# Prompt for the requirepass for redis conf
echo "Please enter the Redis password:"
read -s REDIS_REQUIRE_PASSWORD

# Prompt for the masterauth for redis conf
echo "Please enter the Master Auth password:"
read -s MASTER_AUTH_PASSWORD

# Prompt for the IP address to bind to
echo "Please enter the IP address to bind Redis to:"
read BIND_IP_ADDRESS

# Install the latest stable Redis
curl -fsSL https://packages.redis.io/gpg | sudo gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/redis.list
sudo apt-get update
sudo apt-get install redis

# Add redis service to start on boot
sudo systemctl enable redis

# Add configurations to the end of the file /etc/redis/redis.conf
sed -i "
\$abind $BIND_IP_ADDRESS
\$arequirepass $REDIS_REQUIRE_PASSWORD
\$amasterauth $MASTER_AUTH_PASSWORD
" /etc/redis/redis.conf

echo "This does not add replication. It only adds a master sentinel.
To enable replication add replicaof <masterip> <masterport> to /etc/redis/redis.conf
and restart the redis service.

If you want to add a slave, you want to add the following lines to /etc/redis/redis.conf:
slaveof <masterip> <masterport>
and restart the redis service."

# Restart the redis service
sudo systemctl restart redis

# Test Redis using the redis-cli
redis-cli -a $REDIS_REQUIRE_PASSWORD SET "test" "test"
if redis-cli -a $REDIS_REQUIRE_PASSWORD GET "test" | grep -q "test"; then
  echo "Redis is working!"
else
  echo "Redis is not working!"
fi

# Install sentinel
sed -i "
\$adaemonize yes
\$aport 26379
\$abind $BIND_IP_ADDRESS
\$asupervised systemd
\$apidfile '/run/redis/redis-sentinel.pid'
\$alogfile '/var/log/redis/sentinel.log'
\$asentinel monitor mymaster 127.0.0.1 6379 2
\$asentinel auth-pass mymaster $MASTER_AUTH_PASSWORD
\$asentinel down-after-milliseconds mymaster 5000
\$asentinel failover-timeout mymaster 60000
\$asentinel parallel-syncs mymaster 1
" /etc/redis/sentinel.conf

chown redis:redis /etc/redis/sentinel.conf

# Add sentinel service to start on boot
sed -i "
[Unit]
Description=Redis Sentinel
After=network.target

[Service]
User=redis
Group=redis
Type=notify
ExecStart=/usr/bin/redis-server /etc/redis/sentinel.conf --sentinel
ExecStop=/usr/bin/redis-cli shutdown
Restart=always

[Install]
WantedBy=multi-user.target
" /etc/systemd/system/redis-sentinel.service

# Reload the daemon and start the sentinel service
systemctl daemon-reload
service redis-sentinel start
systemctl enable redis-sentinel

# Test the sentinel
tail -f /var/log/redis/sentinel.log &

if redis-cli -p 26379 info | grep -q "Sentinel"; then
  echo "Sentinel is working!"
else
  echo "Sentinel is not working!"
fi