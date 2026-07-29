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
apk add mosquitto mosquitto-client-ssl kmod-nft-core nftables
```

> **Note:** `jsonfilter` is already built into OpenWrt. `conntrack` needs a kernel module.
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

## Step 3 — Deploy the Subscriber Script

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

## Step 4 — Set Orange Pi Static IP

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

## Step 5 — Update Orange Pi `.env`

Edit `.env` on the Orange Pi:

```env
MQTT_BROKER=tcp://10.0.0.1:1883
MQTT_CLIENT_ID=pisowifi-orangepi
MQTT_USERNAME=pisowifi
MQTT_PASSWORD=<same_password_you_set_in_step_2>
```

---

## Step 6 — Test End-to-End

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
