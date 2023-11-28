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

This is a script to get started with redis-sentinel on Ubuntu.
'

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