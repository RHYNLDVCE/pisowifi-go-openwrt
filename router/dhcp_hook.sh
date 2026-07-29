#!/bin/sh
# dhcp_hook.sh — dnsmasq DHCP lease script for PisoWifi
#
# Called by dnsmasq on every DHCP lease event with:
#   $1 = action  : "add" | "old" | "del"
#   $2 = mac     : client MAC address (e.g. aa:bb:cc:dd:ee:ff)
#   $3 = ip      : client IP address  (e.g. 10.0.1.5)
#   $4 = hostname : (optional)
#
# Deploy:
#   1. Copy this file to /etc/dnsmasq.d/pisowifi_dhcp_hook.sh
#   2. chmod +x /etc/dnsmasq.d/pisowifi_dhcp_hook.sh
#   3. Add to /etc/dnsmasq.conf (or /etc/config/dhcp uci option):
#        dhcp-script=/etc/dnsmasq.d/pisowifi_dhcp_hook.sh
#   4. Restart dnsmasq:
#        /etc/init.d/dnsmasq restart
#
# Dependencies: mosquitto-client (provides mosquitto_pub)
# The mosquitto broker runs on the router itself at 127.0.0.1:1883.

ACTION="$1"
MAC="$2"
IP="$3"
HOSTNAME="$4"

MQTT_HOST="127.0.0.1"
MQTT_PORT="1883"
MQTT_TOPIC="pisowifi/arp"

# Validate inputs
if [ -z "$MAC" ] || [ -z "$IP" ]; then
    exit 0
fi

# Normalise MAC to lowercase
MAC=$(echo "$MAC" | tr '[:upper:]' '[:lower:]')

# Map dnsmasq action → pisowifi action
# "add"  = new lease granted
# "old"  = existing lease renewed / dnsmasq restarted
# "del"  = lease expired or released
case "$ACTION" in
    add|old)
        EVENT="add"
        ;;
    del)
        EVENT="del"
        ;;
    *)
        # Unknown action — ignore
        exit 0
        ;;
esac

PAYLOAD="{\"mac\":\"${MAC}\",\"ip\":\"${IP}\",\"action\":\"${EVENT}\",\"hostname\":\"${HOSTNAME}\"}"

# Publish to MQTT broker (non-blocking, QoS 1, retained so late subscribers get it)
mosquitto_pub \
    -h "$MQTT_HOST" \
    -p "$MQTT_PORT" \
    -t "$MQTT_TOPIC" \
    -q 1 \
    -r \
    -m "$PAYLOAD" \
    2>/dev/null &

logger -t pisowifi "[DHCP] $EVENT $MAC → $IP"
