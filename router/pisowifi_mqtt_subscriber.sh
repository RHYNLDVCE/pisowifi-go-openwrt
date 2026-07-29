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
#   5. /etc/init.d/pisowifi_mqtt enable
#   6. /etc/init.d/pisowifi_mqtt start
#   7. Copy dhcp_hook.sh to /etc/dnsmasq.d/pisowifi_dhcp_hook.sh
#   8. chmod +x /etc/dnsmasq.d/pisowifi_dhcp_hook.sh
#   9. Add 'dhcp-script=/etc/dnsmasq.d/pisowifi_dhcp_hook.sh' to /etc/dnsmasq.conf
#  10. /etc/init.d/dnsmasq restart
#
# Dependencies (install on router):
#   opkg update
#   opkg add mosquitto mosquitto-client-ssl kmod-nft-core nftables tc-full kmod-sched-cake kmod-sched-core kmod-ifb conntrack
#   (jsonfilter is already built into OpenWrt)

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

# Dump the router's ARP table to the Orange Pi via MQTT.
# Called on pisowifi/arp/request — lets the OrangePi rebuild its MAC->IP cache
# after startup or reconnect.
# Each entry is published as a retained pisowifi/arp message so the Orange Pi
# gets them immediately even if it subscribes slightly late.
dump_arp_table() {
    logger -t pisowifi "[ARP] Dumping ARP table to Orange Pi..."
    local count=0
    # /proc/net/arp columns: IP HWtype Flags HWaddr Mask Device
    # Skip the header line, skip incomplete (Flags != 0x2) entries
    while read -r ip hwtype flags mac mask dev; do
        # Skip header row
        [ "$ip" = "IP" ] && continue
        # Skip incomplete/proxy entries (flags must be 0x2 = complete)
        [ "$flags" != "0x2" ] && continue
        # Skip the Orange Pi's own entry and the router itself
        [ "$ip" = "10.0.0.1" ] && continue
        [ "$ip" = "10.0.0.2" ] && continue
        # Normalise MAC to lowercase
        mac=$(echo "$mac" | tr '[:upper:]' '[:lower:]')
        PAYLOAD="{\"mac\":\"${mac}\",\"ip\":\"${ip}\",\"action\":\"add\"}"
        mosquitto_pub \
            -h "$MQTT_HOST" \
            -p "$MQTT_PORT" \
            -t "pisowifi/arp" \
            -q 1 \
            -r \
            -m "$PAYLOAD" \
            2>/dev/null
        count=$(( count + 1 ))
    done < /proc/net/arp
    logger -t pisowifi "[ARP] Dumped $count entries to Orange Pi."
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
    AUTO_PAUSE=$(echo "$payload" | jsonfilter -e '@.auto_pause_enabled')
    TIMEOUT=$(echo "$payload" | jsonfilter -e '@.inactive_timeout')
    BYTES_LIMIT=$(echo "$payload" | jsonfilter -e '@.inactive_bytes_threshold')
    PKTS_LIMIT=$(echo "$payload" | jsonfilter -e '@.inactive_packets_threshold')

    # Save auto-pause config to tmp for the awk monitor
    echo "enabled $AUTO_PAUSE" > /tmp/pisowifi_auto_pause.conf
    echo "timeout $TIMEOUT" >> /tmp/pisowifi_auto_pause.conf
    echo "bytes $BYTES_LIMIT" >> /tmp/pisowifi_auto_pause.conf
    echo "pkts $PKTS_LIMIT" >> /tmp/pisowifi_auto_pause.conf

    logger -t pisowifi "[FIREWALL] Applying nftables init: LAN=$LAN WAN=$WAN"

    # ----------------------------------------------------------------
    # 1. Setup a lightweight uhttpd interceptor on the router.
    # Unauthorized users hit this and get redirected to the portal.
    # The OrangePi identifies the user server-side via c.IP() -> ARP cache,
    # so no MAC/IP injection into the URL is needed.
    # ----------------------------------------------------------------
    mkdir -p /www_portal/cgi-bin
    cat << 'HTM' > /www_portal/cgi-bin/redirect
#!/bin/sh
# Redirect unauthorized users to the captive portal.
# The OrangePi (10.0.0.2) identifies the user from the source IP
# after DNAT preserves it (no hairpin masquerade in nftables).
echo "Status: 302 Found"
echo "Location: http://10.0.0.1/"
echo ""
HTM
    chmod +x /www_portal/cgi-bin/redirect

    # Ensure root path redirects to the CGI script instantly
    cat << 'HTM' > /www_portal/index.html
<html><head><meta http-equiv="refresh" content="0; url=/cgi-bin/redirect"></head><body></body></html>
HTM

    uci delete uhttpd.portal 2>/dev/null || true
    uci set uhttpd.portal=uhttpd
    uci add_list uhttpd.portal.listen_http='0.0.0.0:8080'
    uci set uhttpd.portal.home='/www_portal'
    uci set uhttpd.portal.error_page='/cgi-bin/redirect'
    uci commit uhttpd
    /etc/init.d/uhttpd restart 2>/dev/null || true

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
        
        # Count upload traffic for auto-pause monitor
        iifname "${LAN}" ether saddr @authorized_users
    }

    chain forward_mangle {
        type filter hook forward priority mangle; policy accept;
        tcp flags syn / syn,rst tcp option maxseg size set 1300
    }

    chain postrouting_mangle {
        type filter hook postrouting priority mangle; policy accept;
        ${TTL_RULE}
        
        # Count download traffic for auto-pause monitor
        oifname "${LAN}" ether daddr @authorized_users
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
        
        # Do not intercept the Orange Pi's own internet traffic!
        iifname "${LAN}" ip saddr 10.0.0.2 accept

        # Portal at 10.0.0.1:80 — DNAT to OrangePi.
        # Source IP is preserved (no hairpin masquerade) so c.IP() on OrangePi
        # returns the real client IP. OrangePi uses it to look up MAC from
        # its ARP cache (populated by the DHCP hook via MQTT).
        iifname "${LAN}" ip daddr 10.0.0.1 tcp dport 80 dnat to 10.0.0.2:80

        # Authorized users: send DNS to Cloudflare
        iifname "${LAN}" ether saddr @authorized_users udp dport 53 dnat to 1.1.1.1:53
        iifname "${LAN}" ether saddr @authorized_users tcp dport 53 dnat to 1.1.1.1:53
        
        # Unauthorized users: redirect DNS to router
        iifname "${LAN}" ether saddr != @authorized_users udp dport 53 dnat to 10.0.0.1:53
        iifname "${LAN}" ether saddr != @authorized_users tcp dport 53 dnat to 10.0.0.1:53
        
        # Unauthorized users: intercept HTTP/HTTPS via local router uhttpd (302 → portal)
        iifname "${LAN}" ether saddr != @authorized_users ip daddr != { 10.0.0.1, 10.0.0.3 } tcp dport 80 redirect to :8080
        iifname "${LAN}" ether saddr != @authorized_users ip daddr != { 10.0.0.1, 10.0.0.3 } tcp dport 443 redirect to :8080
    }

    chain nat_postrouting {
        type nat hook postrouting priority srcnat; policy accept;
        
        # NOTE: There is intentionally NO masquerade rule for LAN→OrangePi traffic.
        # The old hairpin masquerade was removed so that the OrangePi can see the
        # real client source IP via c.IP(). The OrangePi uses policy routing to
        # force its HTTP replies back through the router, where conntrack applies
        # reverse DNAT transparently. See cmd/server/main.go:setupPolicyRouting().
        
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
# Auto-Pause Monitor Loop
# ---------------------------------------------------------------------------
# Periodically reads nftables counters and uses an awk state machine to
# track idle users. If a user is inactive for the timeout, it sends a pause command.
auto_pause_monitor() {
    logger -t pisowifi "[MQTT] Auto-pause monitor started."
    while true; do
        sleep 15
        echo "MARKER_START $(date +%s)"
        nft list set ip pisowifi authorized_users 2>/dev/null | grep -oE '([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2} counter packets [0-9]+ bytes [0-9]+'
    done | awk -v host="$MQTT_HOST" -v port="$MQTT_PORT" '
    /^MARKER_START/ {
        now = $2
        
        # Read config dynamically
        enabled = "false"
        timeout = 0
        bytes_limit = 0
        pkts_limit = 0
        while ((getline < "/tmp/pisowifi_auto_pause.conf") > 0) {
            if ($1 == "enabled") enabled = $2
            if ($1 == "timeout") timeout = $2 + 0
            if ($1 == "bytes") bytes_limit = $2 + 0
            if ($1 == "pkts") pkts_limit = $2 + 0
        }
        close("/tmp/pisowifi_auto_pause.conf")

        if (enabled == "true" && timeout > 0) {
            for (mac in last_active) {
                # Ensure the user is still authorized in this cycle (not manually paused/removed)
                if (!(mac in seen)) {
                    delete last_active[mac]
                    delete last_bytes[mac]
                    delete last_pkts[mac]
                    continue
                }

                if ((now - last_active[mac]) > timeout) {
                    printf "mosquitto_pub -h %s -p %s -t pisowifi/pause_user -m '\''{\"mac\":\"%s\"}'\''\n", host, port, mac
                    delete last_active[mac]
                    delete last_bytes[mac]
                    delete last_pkts[mac]
                }
            }
        }
        delete seen
        next
    }
    /counter packets/ {
        mac = $1; pkts = $4; bytes = $6
        seen[mac] = 1
        if (!(mac in last_bytes)) {
            last_bytes[mac] = bytes
            last_pkts[mac] = pkts
            last_active[mac] = now
        } else {
            diff_b = bytes - last_bytes[mac]
            diff_p = pkts - last_pkts[mac]
            last_bytes[mac] = bytes
            last_pkts[mac] = pkts
            if (diff_b > bytes_limit || diff_p > pkts_limit) {
                last_active[mac] = now
            }
        }
    }' | sh 2>/dev/null
}

# Start monitor in the background
auto_pause_monitor &

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
                nft add element ip pisowifi authorized_users "{ $MAC counter }" 2>/dev/null || true
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
        # pisowifi/arp/request
        # Orange Pi sends this on startup or reconnect to ask the router
        # to dump its full ARP table (MAC→IP) as retained pisowifi/arp
        # messages so the OrangePi can rebuild its in-memory cache.
        # ----------------------------------------------------------------
        "pisowifi/arp/request")
            dump_arp_table
            ;;

        # ----------------------------------------------------------------
        # pisowifi/lwt
        # ----------------------------------------------------------------
        "pisowifi/lwt")
            status=$(echo "$payload" | jsonfilter -e '@.status' 2>/dev/null)
            if [ "$status" = "offline" ]; then
                logger -t pisowifi "[FAILSAFE] Orange Pi offline! Flushing all connections and blocking users."
                nft flush set ip pisowifi authorized_users 2>/dev/null || true
                conntrack -F 2>/dev/null || true
            elif [ "$status" = "online" ]; then
                logger -t pisowifi "[FAILSAFE] Orange Pi is back online."
            fi
            ;;

        # ----------------------------------------------------------------
        # pisowifi/router/stats/request
        # ----------------------------------------------------------------
        "pisowifi/router/stats/request")
            # Run in subshell so we don't block the MQTT loop
            (
                # Uptime
                read -r uptime_sec _ < /proc/uptime
                uptime_sec=${uptime_sec%.*}

                # Memory
                mem_total=$(awk '/MemTotal/ {print $2}' /proc/meminfo)
                mem_avail=$(awk '/MemAvailable/ {print $2}' /proc/meminfo)

                # Disk Storage (Root overlay in 1K blocks)
                disk_total=$(df / | awk 'NR==2 {print $2}')
                disk_free=$(df / | awk 'NR==2 {print $4}')

                # CPU Load (1 min average)
                read -r load1 _ < /proc/loadavg

                # Temperature
                temp_raw=$(cat /sys/class/thermal/thermal_zone0/temp 2>/dev/null || echo "")
                if [ -n "$temp_raw" ]; then
                    temp_c=$(awk -v t="$temp_raw" 'BEGIN {printf "%.1f", t/1000}')
                else
                    temp_c="N/A"
                fi

                # Interfaces (find the active WAN and LAN)
                # LAN is usually br-lan, WAN might be eth0 or wan
                LAN_IF=$(nft list tables | grep -q pisowifi && nft list chain ip pisowifi filter_forward 2>/dev/null | grep iifname | head -1 | awk '{print $2}' | tr -d '"' || echo "br-lan")
                WAN_IF=$(ip route | awk '/default/ {print $5}')
                
                # Get IP addresses, comma separated then replaced to literal \n for JSON
                ips_str=$(ip -4 addr show | grep inet | grep -v '127.0.0.1' | awk '{print $2}' | cut -d/ -f1 | paste -sd, - | sed 's/,/\\n/g')

                wan_rx=$(grep "${WAN_IF}:" /proc/net/dev | awk '{print $2}')
                wan_tx=$(grep "${WAN_IF}:" /proc/net/dev | awk '{print $10}')
                lan_rx=$(grep "${LAN_IF}:" /proc/net/dev | awk '{print $2}')
                lan_tx=$(grep "${LAN_IF}:" /proc/net/dev | awk '{print $10}')

                PAYLOAD="{\"uptime\":${uptime_sec:-0},\"mem_total\":${mem_total:-0},\"mem_avail\":${mem_avail:-0},\"disk_total\":${disk_total:-0},\"disk_free\":${disk_free:-0},\"load\":\"${load1:-0}\",\"temp\":\"${temp_c}\",\"ips\":\"${ips_str}\",\"wan_rx\":${wan_rx:-0},\"wan_tx\":${wan_tx:-0},\"lan_rx\":${lan_rx:-0},\"lan_tx\":${lan_tx:-0}}"
                mosquitto_pub -h "$MQTT_HOST" -p "$MQTT_PORT" -t "pisowifi/router/stats/reply" -m "$PAYLOAD" 2>/dev/null
            ) &
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
