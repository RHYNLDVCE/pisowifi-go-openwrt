#!/bin/bash
# fail_safe.sh
# Triggered by systemd when the pisowifi-server stops or crashes.

ROUTER_IP="10.0.0.1"

# 1. Send the "offline" Last Will message to the router
# The router's MQTT subscriber will catch this, flush all connections,
# and block all active users instantly to prevent free internet.
mosquitto_pub -h $ROUTER_IP -t 'pisowifi/lwt' -m '{"status":"offline"}' 2>/dev/null

# 2. Turn off the Coin Slot Relay on the Orange Pi 3 LTS (GPIO 121)
# This uses sysfs to ensure power is cut to the coin acceptor so it rejects coins
if [ -d "/sys/class/gpio/gpio121" ]; then
    echo "0" > /sys/class/gpio/gpio121/value 2>/dev/null
fi

echo "🔒 FAIL-SAFE TRIGGERED: Sent offline signal to router & disabled coin slot."