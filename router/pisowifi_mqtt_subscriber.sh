#!/bin/sh
# pisowifi_mqtt_subscriber.sh
#
# Router-side MQTT subscriber for PisoWifi.
# Runs on the OpenWrt router (10.0.0.1).
#
# Listens for commands published by the Orange Pi and applies them locally
# using nftables (nft), tc, and conntrack.
#
# Deploy:
#   1. Copy this file to /usr/bin/pisowifi_mqtt_subscriber.sh
#   2. chmod +x /usr/bin/pisowifi_mqtt_subscriber.sh
#   3. Copy pisowifi_mqtt.init to /etc/init.d/pisowifi_mqtt
#   4. chmod +x /etc/init.d/pisowifi_mqtt
#   5. rc-update add pisowifi_mqtt
#   6. rc-service pisowifi_mqtt start
#
# Dependencies (install on router):
#   apk update
#   apk add mosquitto mosquitto-client-ssl kmod-nft-core nftables tc-full kmod-sched-cake kmod-sched-core kmod-ifb
#   (conntrack and jsonfilter are already built into OpenWrt)

MQTT_HOST="10.0.0.1"
MQTT_PORT="1883"
MQTT_TOPICS="pisowifi/#"

# UID helper: derives a unique integer from an IP address for tc class IDs.
# Mirrors getUID() in the Orange Pi Go code.
get_uid() {
    local ip="$1"
    local c d
    c=$(echo "$ip" | awk -F. '{print $3}')
    d=$(echo "$ip" | awk -F. '{print $4}')
    echo $(( c * 256 + d ))
}

# Apply full nftables ruleset from a JSON payload.
# Called on pisowifi/firewall/init and pisowifi/firewall/reload
apply_firewall_init() {
    local payload="$1"

    LAN=$(echo "$payload"       | jsonfilter -e '@.lan')
    WAN=$(echo "$payload"       | jsonfilter -e '@.wan')
    CUSTOM_TTL=$(echo "$payload" | jsonfilter -e '@.custom_ttl')
    UDP_PRIO=$(echo "$payload"  | jsonfilter -e '@.udp_priority')
    OPEN_NAT=$(echo "$payload"  | jsonfilter -e '@.open_nat')
    SQM=$(echo "$payload"       | jsonfilter -e '@.sqm_enabled')
    SQM_UP=$(echo "$payload"    | jsonfilter -e '@.sqm_upload_mbps')
    SQM_DOWN=$(echo "$payload"  | jsonfilter -e '@.sqm_download_mbps')

    logger -t pisowifi "[FIREWALL] Applying nftables init: LAN=$LAN WAN=$WAN"

    # Kernel tuning
    sysctl -w net.core.default_qdisc=fq
    sysctl -w net.ipv4.tcp_congestion_control=bbr
    sysctl -w net.ipv4.ip_forward=1
    sysctl -w net.ipv6.conf.all.disable_ipv6=1
    sysctl -w net.ipv6.conf.default.disable_ipv6=1
    sysctl -w net.netfilter.nf_conntrack_max=65536
    sysctl -w net.netfilter.nf_conntrack_tcp_timeout_established=1800
    sysctl -w net.netfilter.nf_conntrack_udp_timeout=30
    sysctl -w net.netfilter.nf_conntrack_udp_timeout_stream=60
    sysctl -w net.ipv4.tcp_fastopen=3
    sysctl -w net.ipv4.tcp_tw_reuse=1
    sysctl -w net.ipv4.tcp_fin_timeout=15
    sysctl -w net.ipv4.ip_local_port_range="1024 65535"

    # Open NAT (miniupnpd)
    if [ "$OPEN_NAT" = "true" ]; then
        /etc/init.d/miniupnpd restart 2>/dev/null || true
    else
        /etc/init.d/miniupnpd stop 2>/dev/null || true
    fi

    # Flush and rebuild nftables
    nft flush ruleset

    # Build UDP priority rules string
    UDP_PRIO_RULES=""
    if [ "$UDP_PRIO" = "true" ]; then
        UDP_PRIO_RULES='
        udp sport { 5000-5500, 7074-7750, 10000-10009, 30000-30300 } meta mark set 0x63
        udp dport { 5000-5500, 7074-7750, 10000-10009, 30000-30300 } meta mark set 0x63
        meta mark 0x63 ip dscp set cs4
        udp length <= 256 meta mark set 0x63
        udp length <= 256 ip dscp set ef
        tcp dport 6881-6889 ip dscp set cs1
        udp dport 6881-6889 ip dscp set cs1'
    fi

    # Build TTL rule
    TTL_RULE=""
    if [ -n "$CUSTOM_TTL" ] && [ "$CUSTOM_TTL" != "0" ]; then
        TTL_RULE="oifname \"${LAN}\" ip ttl set ${CUSTOM_TTL}"
    fi

    # Apply nftables ruleset
    nft -f - <<EOF
# Fix miniupnpd aggressive drop on forward
add table inet filter
add chain inet filter forward { type filter hook forward priority filter; policy accept; }

add table ip pisowifi
flush table ip pisowifi
table ip pisowifi {
    set authorized_users {
        type ether_addr
        size 65535
        flags dynamic
        counter
    }

    chain prerouting {
        type filter hook prerouting priority mangle; policy accept;
        ${UDP_PRIO_RULES}
    }

    chain forward_mangle {
        type filter hook forward priority mangle; policy accept;
        tcp flags syn / syn,rst tcp option maxseg size set 1300
    }

    chain postrouting_mangle {
        type filter hook postrouting priority mangle; policy accept;
        ${TTL_RULE}
    }

    chain filter_input {
        type filter hook input priority filter; policy accept;
        ct state related,established accept
        iifname "${LAN}" udp sport 67-68 udp dport 67-68 accept
        iifname "lo" accept
    }

    chain filter_forward {
        type filter hook forward priority filter; policy drop;
        ct state established,related accept
        
        # 1. Allow the Orange Pi (10.0.0.2) full internet access
        iifname "${LAN}" ip saddr 10.0.0.2 accept
        oifname "${LAN}" ip daddr 10.0.0.2 accept

        # 2. Allow authorized users
        iifname "${LAN}" ether saddr @authorized_users accept
        oifname "${LAN}" ether daddr @authorized_users accept
        
        # 3. Allow DNS for everyone
        iifname "${LAN}" udp dport 53 accept
        iifname "${LAN}" tcp dport 53 accept
        
        # 4. Drop HTTPS for unauthorized users (prevents timeouts)
        iifname "${LAN}" ether saddr != @authorized_users tcp dport 443 drop
    }

    chain nat_prerouting {
        type nat hook prerouting priority dstnat; policy accept;
        
        # Do not intercept the Orange Pi's own traffic
        iifname "${LAN}" ip saddr 10.0.0.2 accept
        
        # Make the portal explicitly accessible at 10.0.0.1 for everyone (auth & unauth)
        iifname "${LAN}" ip daddr 10.0.0.1 tcp dport 80 dnat to 10.0.0.2:80
        
        # Authorized users: send DNS to Cloudflare
        iifname "${LAN}" ether saddr @authorized_users udp dport 53 dnat to 1.1.1.1:53
        iifname "${LAN}" ether saddr @authorized_users tcp dport 53 dnat to 1.1.1.1:53
        
        # Unauthorized users: redirect DNS to router, HTTP/HTTPS to Orange Pi portal
        iifname "${LAN}" ether saddr != @authorized_users udp dport 53 dnat to 10.0.0.1:53
        iifname "${LAN}" ether saddr != @authorized_users tcp dport 53 dnat to 10.0.0.1:53
        iifname "${LAN}" ether saddr != @authorized_users tcp dport 80 dnat to 10.0.0.2:80
        iifname "${LAN}" ether saddr != @authorized_users tcp dport 443 dnat to 10.0.0.2:80
    }

    chain nat_postrouting {
        type nat hook postrouting priority srcnat; policy accept;
        oifname "${WAN}" masquerade
    }
}
EOF

    # TC setup on LAN interface
    tc qdisc del dev "$LAN" root 2>/dev/null || true
    tc qdisc del dev "$LAN" ingress 2>/dev/null || true
    tc qdisc add dev "$LAN" root handle 1: htb default 10
    tc class add dev "$LAN" parent 1: classid 1:ffff htb rate 1000mbit
    tc qdisc add dev "$LAN" ingress

    # SQM on WAN
    tc qdisc del dev "$WAN" root 2>/dev/null || true
    tc qdisc del dev "$WAN" ingress 2>/dev/null || true
    ip link del ifb0 2>/dev/null || true

    if [ "$SQM" = "true" ]; then
        tc qdisc add dev "$WAN" root cake bandwidth "${SQM_UP}mbit" diffserv4 nat wash
        ip link add name ifb0 type ifb
        ip link set dev ifb0 up
        tc qdisc add dev "$WAN" handle ffff: ingress
        tc filter add dev "$WAN" parent ffff: protocol all u32 match u32 0 0 action mirred egress redirect dev ifb0
        tc qdisc add dev ifb0 root cake bandwidth "${SQM_DOWN}mbit" diffserv4 nat wash
        logger -t pisowifi "[FIREWALL] SQM enabled: up=${SQM_UP}mbit down=${SQM_DOWN}mbit"
    fi

    # Flush conntrack to clear stale sessions after rule rebuild
    conntrack -F 2>/dev/null || true

    logger -t pisowifi "[FIREWALL] nftables init complete."
}

# ---------------------------------------------------------------------------
# Main subscriber loop
# ---------------------------------------------------------------------------
logger -t pisowifi "[MQTT] Subscriber starting on topics: $MQTT_TOPICS"

mosquitto_sub \
    -h "$MQTT_HOST" \
    -p "$MQTT_PORT" \
    -t "$MQTT_TOPICS" \
    --nodelay \
    -v \
| while IFS= read -r line; do
    topic=$(echo "$line" | awk '{print $1}')
    payload=$(echo "$line" | cut -d' ' -f2-)

    case "$topic" in

        # ----------------------------------------------------------------
        # pisowifi/allow {"mac":"aa:bb:cc:dd:ee:ff","ip":"10.0.1.5"}
        # ----------------------------------------------------------------
        "pisowifi/allow")
            MAC=$(echo "$payload" | jsonfilter -e '@.mac')
            IP=$(echo "$payload"  | jsonfilter -e '@.ip')
            if [ -n "$MAC" ]; then
                nft add element ip pisowifi authorized_users "{ $MAC }" 2>/dev/null || true
                logger -t pisowifi "[ALLOW] $MAC ($IP)"
            fi
            ;;

        # ----------------------------------------------------------------
        # pisowifi/block {"mac":"aa:bb:cc:dd:ee:ff","ip":"10.0.1.5"}
        # ----------------------------------------------------------------
        "pisowifi/block")
            MAC=$(echo "$payload" | jsonfilter -e '@.mac')
            IP=$(echo "$payload"  | jsonfilter -e '@.ip')
            if [ -n "$MAC" ]; then
                nft delete element ip pisowifi authorized_users "{ $MAC }" 2>/dev/null || true
            fi
            if [ -n "$IP" ]; then
                conntrack -D -s "$IP" 2>/dev/null || true
                conntrack -D -d "$IP" 2>/dev/null || true
            fi
            logger -t pisowifi "[BLOCK] $MAC ($IP)"
            ;;

        # ----------------------------------------------------------------
        # pisowifi/speed_limit/apply {"ip":"10.0.1.5","mbps":5}
        # ----------------------------------------------------------------
        "pisowifi/speed_limit/apply")
            IP=$(echo "$payload"  | jsonfilter -e '@.ip')
            MBPS=$(echo "$payload" | jsonfilter -e '@.mbps')
            LAN=$(nft list tables | grep -q pisowifi && nft list chain ip pisowifi filter_forward 2>/dev/null | grep iifname | head -1 | awk '{print $2}' | tr -d '"' || echo "br-lan")

            if [ -n "$IP" ] && [ -n "$MBPS" ]; then
                UID=$(get_uid "$IP")
                UID_HEX=$(printf '%x' "$UID")
                SPEED="${MBPS}mbit"
                UPLOAD_KBPS=$(( MBPS * 1024 ))

                tc class replace dev "$LAN" parent 1:ffff classid "1:${UID_HEX}" htb rate "$SPEED" ceil "$SPEED" burst 15k cburst 15k
                tc qdisc replace dev "$LAN" parent "1:${UID_HEX}" handle "${UID_HEX}:" cake bandwidth "$SPEED"
                tc filter add dev "$LAN" protocol ip parent 1:0 prio "$UID" u32 match ip dst "$IP" flowid "1:${UID_HEX}"
                tc filter add dev "$LAN" parent ffff: protocol ip prio "$UID" u32 match ip src "$IP" police rate "${UPLOAD_KBPS}kbit" burst 12k drop flowid :1
                logger -t pisowifi "[SPEED] Applied ${MBPS}mbit to $IP"
            fi
            ;;

        # ----------------------------------------------------------------
        # pisowifi/speed_limit/remove {"ip":"10.0.1.5"}
        # ----------------------------------------------------------------
        "pisowifi/speed_limit/remove")
            IP=$(echo "$payload" | jsonfilter -e '@.ip')
            LAN=$(nft list tables | grep -q pisowifi && echo "br-lan" || echo "br-lan")

            if [ -n "$IP" ]; then
                UID=$(get_uid "$IP")
                UID_HEX=$(printf '%x' "$UID")
                tc filter del dev "$LAN" protocol ip parent 1:0 prio "$UID" 2>/dev/null || true
                tc filter del dev "$LAN" protocol ip parent ffff: prio "$UID" 2>/dev/null || true
                tc qdisc del dev "$LAN" parent "1:${UID_HEX}" 2>/dev/null || true
                tc class del dev "$LAN" parent 1:ffff classid "1:${UID_HEX}" 2>/dev/null || true
                logger -t pisowifi "[SPEED] Removed limit for $IP"
            fi
            ;;

        # ----------------------------------------------------------------
        # pisowifi/firewall/init  — full init (called at Orange Pi startup)
        # pisowifi/firewall/reload — reload after config change
        # ----------------------------------------------------------------
        "pisowifi/firewall/init"|"pisowifi/firewall/reload")
            apply_firewall_init "$payload"
            ;;

        *)
            # Unknown topic — ignore silently
            ;;
    esac
done
