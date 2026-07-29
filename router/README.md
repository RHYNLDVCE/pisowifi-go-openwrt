# PisoWifi Router Setup Guide (OpenWrt)

This directory contains all files that need to be deployed to your **OpenWrt router**.
The Orange Pi runs the Go server and React UI; the router enforces nftables/tc rules
via MQTT commands from the Orange Pi.

---

## Network IP Plan

| Device | IP | Role |
|---|---|---|
| Router LAN | `10.0.0.1` | Default gateway, MQTT broker, nftables, DHCP |
| Orange Pi | `10.0.0.2` (set static) | Go server, React admin UI, portal `:80` |
| User devices | `10.0.x.x` | DHCP from router pool |

> **Captive portal**: Unauthenticated users' port 80/443 requests are DNATed by the router to `10.0.0.2:80` (Orange Pi). The Orange Pi serves the portal page.

---

## Step 1 — Install Packages on the Router

SSH into your router and run:

```sh
apk update
apk add mosquitto mosquitto-client-ssl kmod-nft-core nftables tc-full kmod-sched-cake kmod-sched-core kmod-ifb conntrack
```

> **Note:** `jsonfilter` is already built into OpenWrt.
> Run:
> ```sh
> apk add kmod-nf-conntrack
> ```
> Then verify the userspace command is available:
> ```sh
> which conntrack   # → /usr/sbin/conntrack  (if available)
> ```
> If `conntrack` command is still missing after installing the module, the subscriber script
> handles this gracefully — block rules still apply via nftables; conntrack flush just won't run.

---

## Step 2 — Configure Mosquitto

```sh
# Create mosquitto directories
mkdir -p /etc/mosquitto /tmp/mosquitto

# Copy config (run this from your laptop)
scp mosquitto.conf root@10.0.0.1:/etc/mosquitto/mosquitto.conf

# On the router — create the pisowifi MQTT user password
mosquitto_passwd -c /etc/mosquitto/passwd pisowifi

# Enable and start mosquitto
rc-service mosquitto enable
rc-service mosquitto start
```

> ⚠️ **The password you set here must match `MQTT_PASSWORD` in the Orange Pi's `.env` file and in `pisowifi_mqtt_subscriber.sh`.**

---

## Step 3 — Deploy the DHCP Hook (MAC→IP Sync)

This script is called by dnsmasq on every DHCP lease event and publishes the
client's MAC and IP to the MQTT broker so the Orange Pi always knows who is online.

```sh
# Copy the hook script to the router
scp dhcp_hook.sh root@10.0.0.1:/etc/dnsmasq.d/pisowifi_dhcp_hook.sh

# Make it executable
ssh root@10.0.0.1 "chmod +x /etc/dnsmasq.d/pisowifi_dhcp_hook.sh"

# Tell dnsmasq to call it on every lease event
# (adds the dhcp-script option to dnsmasq.conf)
ssh root@10.0.0.1 "echo 'dhcp-script=/etc/dnsmasq.d/pisowifi_dhcp_hook.sh' >> /etc/dnsmasq.conf"

# Restart dnsmasq to apply
ssh root@10.0.0.1 "/etc/init.d/dnsmasq restart"
```

> ✅ From this point on, whenever any client gets a DHCP lease the router will
> publish `pisowifi/arp {"mac":"...","ip":"...","action":"add"}` to the MQTT broker.
> The Orange Pi subscribes to this topic and updates its in-memory ARP cache in real time.

---

## Step 4 — Deploy the Subscriber Script

```sh
# Copy the subscriber script
scp pisowifi_mqtt_subscriber.sh root@10.0.0.1:/usr/bin/pisowifi_mqtt_subscriber.sh

# Make it executable
ssh root@10.0.0.1 "chmod +x /usr/bin/pisowifi_mqtt_subscriber.sh"

# Copy the init.d service
scp pisowifi_mqtt.init root@10.0.0.1:/etc/init.d/pisowifi_mqtt

# Make it executable, add to startup, and start it
ssh root@10.0.0.1 "chmod +x /etc/init.d/pisowifi_mqtt && rc-update add pisowifi_mqtt && rc-service pisowifi_mqtt start"
```

---

## Step 5 — Set Orange Pi Static IP

On the router's DHCP server, assign `10.0.0.2` as a static lease for the Orange Pi's MAC address.

In OpenWrt LuCI: **Network → DHCP and DNS → Static Leases → Add**

Or via UCI:
```sh
uci add dhcp host
uci set dhcp.@host[-1].mac="AA:BB:CC:DD:EE:FF"   # ← your Orange Pi MAC
uci set dhcp.@host[-1].ip="10.0.0.2"
uci set dhcp.@host[-1].name="orangepi"
uci commit dhcp
rc-service dnsmasq restart
```

---

## Step 6 — Update Orange Pi `.env`

Edit `.env` on the Orange Pi:

```env
MQTT_BROKER=tcp://10.0.0.1:1883
MQTT_CLIENT_ID=pisowifi-orangepi
MQTT_USERNAME=pisowifi
MQTT_PASSWORD=<same_password_you_set_in_step_2>
```

---

## Step 7 — Test End-to-End

### On the router, watch MQTT traffic:
```sh
mosquitto_sub -h localhost -u pisowifi -P <password> -t 'pisowifi/#' -v
```

### Start the Orange Pi Go server. In the router terminal you should see:
```
pisowifi/firewall/init  {"lan":"br-lan","wan":"eth0","custom_ttl":1,...}
```

### Verify nftables was applied:
```sh
nft list set ip pisowifi authorized_users
```

### Watch the ARP sync:
```sh
# On the router — watch DHCP events arrive as pisowifi/arp messages
mosquitto_sub -h localhost -t 'pisowifi/arp' -v
# Expected on each device connect:
# pisowifi/arp {"mac":"aa:bb:cc:dd:ee:ff","ip":"10.0.1.5","action":"add"}
```

### Insert a coin on a connected device and watch:
```
pisowifi/allow  {"mac":"aa:bb:cc:dd:ee:ff","ip":"10.0.1.5"}
```

### Verify the MAC appeared:
```sh
nft list set ip pisowifi authorized_users
```

---

## Troubleshooting

| Problem | Fix |
|---|---|
| `[MQTT] Initial connect failed` on Orange Pi | Check Mosquitto is running: `ssh root@10.0.0.1 "rc-service mosquitto status"` |
| `nft: Operation not permitted` | Make sure `kmod-nft-core` and `nftables` are installed: `apk add kmod-nft-core nftables` |
| `jsonfilter: command not found` | `apk add jsonfilter` |
| Users can't reach portal | Check DNAT rule in `nat_prerouting` chain — ensure `10.0.0.2` is reachable from LAN |
| Subscriber dies after router reboot | Check `rc-update add pisowifi_mqtt` was run |
| Orange Pi shows wrong/no IP for users | Check DHCP hook is installed and dnsmasq restarted: `logread \| grep pisowifi` |
| No `pisowifi/arp` messages on reconnect | Orange Pi publishes `pisowifi/arp/request` — check router subscriber is running |
