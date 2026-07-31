#!/bin/sh
# PisoWifi Router Dependency Installer
# Run this directly on your OpenWrt router via SSH to install required packages.

echo "=========================================="
echo "    PisoWifi OpenWrt Router Installer     "
echo "=========================================="

echo "Updating apk package list..."
apk update

echo ""
echo "Installing required networking and MQTT packages..."
# Install mosquitto for MQTT
# Install nftables for firewall manipulation
# Install tc-full & sched-cake for SQM/speed limiting
# Install conntrack to flush active sessions
apk add \
    mosquitto \
    mosquitto-client-nossl \
    kmod-nft-core \
    nftables \
    tc-full \
    kmod-sched-cake \
    kmod-sched-core \
    kmod-ifb \
    conntrack \
    kmod-nf-conntrack

echo ""
echo "Configuring LAN Subnet to 10.0.0.1/16..."
uci set network.lan.ipaddr='10.0.0.1'
uci set network.lan.netmask='255.255.0.0'

echo "Adding secondary Admin IP (10.0.0.3) for LuCI access..."
uci set network.lan_admin=interface
uci set network.lan_admin.proto='static'
uci set network.lan_admin.device='br-lan'
uci set network.lan_admin.ipaddr='10.0.0.3'
uci set network.lan_admin.netmask='255.255.255.0'

uci commit network

echo ""
echo "Disabling IPv6 on the LAN to prevent captive portal bypass..."
uci set network.lan.ipv6='0'
uci delete network.lan.ip6assign 2>/dev/null || true
uci commit network

echo ""
echo "Configuring DHCP pool for captive portal..."
uci set dhcp.lan.start='10'
uci set dhcp.lan.limit='60000'
uci set dhcp.lan.leasetime='12h'

# Disable IPv6 DHCP/RA features
uci set dhcp.lan.dhcpv6='disabled'
uci set dhcp.lan.ra='disabled'
uci set dhcp.lan.ndp='disabled'

uci commit dhcp

echo ""
echo "=========================================="
echo "    Orange Pi Static IP Setup"
echo "=========================================="
echo "To ensure the Orange Pi always gets 10.0.0.2, we need its MAC address."
echo "If you know the Orange Pi MAC address (e.g. AA:BB:CC:DD:EE:FF), type it below."
echo "If you want to do this later via LuCI, just press Enter to skip."
printf "Orange Pi MAC Address: "
read -r OPI_MAC

if [ -n "$OPI_MAC" ]; then
    echo "Binding $OPI_MAC to 10.0.0.2..."
    # Clear any existing static lease for 10.0.0.2 to prevent dnsmasq crash loops
    while uci -q delete dhcp.@host[0]; do :; done
    uci add dhcp host
    uci set dhcp.@host[-1].mac="$OPI_MAC"
    uci set dhcp.@host[-1].ip="10.0.0.2"
    uci set dhcp.@host[-1].name="orangepi"
    uci commit dhcp
    echo "Static lease saved!"
else
    echo "Skipping static IP setup."
fi

echo ""
echo "Setting up Mosquitto Config..."
if [ -f "./mosquitto.conf" ]; then
    mkdir -p /etc/mosquitto
    cp ./mosquitto.conf /etc/mosquitto/mosquitto.conf
fi

echo "Ensuring Mosquitto MQTT broker starts on boot..."
/etc/init.d/mosquitto enable
/etc/init.d/mosquitto restart

echo ""
echo "Setting up the DHCP Hook (ARP sync)..."
if [ -f "./dhcp_hook.sh" ]; then
    mkdir -p /etc/dnsmasq.d
    cp ./dhcp_hook.sh /etc/dnsmasq.d/pisowifi_dhcp_hook.sh
    chmod +x /etc/dnsmasq.d/pisowifi_dhcp_hook.sh
    grep -q 'dhcp-script=/etc/dnsmasq.d/pisowifi_dhcp_hook.sh' /etc/dnsmasq.conf || echo 'dhcp-script=/etc/dnsmasq.d/pisowifi_dhcp_hook.sh' >> /etc/dnsmasq.conf
    /etc/init.d/dnsmasq restart
fi

echo ""
echo "Setting up the PisoWifi MQTT Subscriber Service..."
if [ -f "./pisowifi_mqtt_subscriber.sh" ] && [ -f "./pisowifi_mqtt.init" ]; then
    echo "Copying scripts to system directories..."
    cp ./pisowifi_mqtt_subscriber.sh /usr/bin/
    chmod +x /usr/bin/pisowifi_mqtt_subscriber.sh
    
    cp ./pisowifi_mqtt.init /etc/init.d/pisowifi_mqtt
    chmod +x /etc/init.d/pisowifi_mqtt
    
    echo "Enabling and starting the background service..."
    /etc/init.d/pisowifi_mqtt enable
    /etc/init.d/pisowifi_mqtt start
    echo "Service is now running!"
else
    echo "Warning: 'pisowifi_mqtt_subscriber.sh' or 'pisowifi_mqtt.init' not found in this folder."
    echo "Please make sure they are in the same folder as this script so they can be installed automatically."
fi

echo "=========================================="
echo " Router Installation & Configuration Complete!"
echo " IMPORTANT: The router IP has been set to 10.0.0.1."
echo " Please reboot the router, or run '/etc/init.d/network restart'."
echo " Note: Your current SSH connection will drop when you restart the network."
echo "=========================================="
