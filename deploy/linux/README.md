# VALDR Testnet Linux deployment

Reference platform: Ubuntu Server 24.04 LTS, x86_64. ARM64 is not a release target until the dedicated CI gate exists.

## Layout

- binaries: `/usr/local/bin/valdrd`, `valdr-cli`, `valdr-miner`, `valdr-explorer`
- node and Explorer state: `/var/lib/valdr`
- operator configuration/private local files: `/etc/valdr`
- service account: `valdr` with no interactive shell
- P2P: TCP 17333
- RPC: TCP 17332 bound to localhost
- Explorer: TCP 8080 bound to localhost; expose only through an HTTPS reverse proxy

Testnet VDR has no promised monetary value.

## Install

```bash
sudo useradd --system --home /var/lib/valdr --create-home --shell /usr/sbin/nologin valdr
sudo install -d -o valdr -g valdr -m 0700 /var/lib/valdr /etc/valdr
printf 'VALDR_ADVERTISE_ADDRESS=node.example.org:17333\n' | sudo tee /etc/valdr/valdr.env >/dev/null
sudo chown root:valdr /etc/valdr/valdr.env
sudo chmod 0640 /etc/valdr/valdr.env
sudo install -o root -g root -m 0755 valdrd valdr-cli valdr-miner valdr-explorer /usr/local/bin/
sudo install -o root -g root -m 0644 deploy/systemd/valdrd.service /etc/systemd/system/valdrd.service
sudo install -o root -g root -m 0644 deploy/systemd/valdr-explorer.service /etc/systemd/system/valdr-explorer.service
sudo systemctl daemon-reload
sudo systemctl enable --now valdrd
sudo systemctl enable --now valdr-explorer
```

Logs go to stdout/stderr and are collected by journald:

```bash
journalctl -u valdrd -f
journalctl -u valdr-explorer -f
```

## Firewall and reverse proxy

Set `VALDR_ADVERTISE_ADDRESS` to the real routable DNS name or public IP plus `:17333`. Do not use `0.0.0.0`, loopback, private, multicast or link-local addresses for a public Testnet advertisement.

Expose only P2P from the node host:

```bash
sudo ufw allow 17333/tcp
sudo ufw deny 17332/tcp
sudo ufw deny 8080/tcp
```

Keep RPC on `127.0.0.1:17332`. Keep Explorer on `127.0.0.1:8080` and terminate public TLS at a reverse proxy. Do not proxy node RPC to the Internet.

Minimal nginx location after TLS has been configured:

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto https;
}
```

## Verify

```bash
valdrd status --node http://127.0.0.1:17332
sudo -u valdr valdrd verify-db --data /var/lib/valdr --network testnet2
curl --fail http://127.0.0.1:8080/healthz
```

Expected chain ID: `valdr-testnet-2`.

## Backup and restore

Stop services before a cold backup so BadgerDB and the Explorer index are captured consistently:

```bash
sudo systemctl stop valdr-explorer valdrd
sudo tar -C /var/lib -czf valdr-testnet-backup.tgz valdr
sudo systemctl start valdrd valdr-explorer
```

Restore into an empty `/var/lib/valdr`, restore ownership to `valdr:valdr`, then run `valdrd verify-db --network testnet2` before enabling the services. The Explorer index is derived data and may be deleted and rebuilt if necessary.

## Upgrade and rollback

1. Back up `/var/lib/valdr`.
2. Stop Explorer and node.
3. Save the current binaries as a versioned rollback set.
4. Install the new signed/reviewed binaries.
5. Start `valdrd`, run `verify-db`, then start Explorer.
6. Confirm chain ID, tip, peer count, chainwork and Explorer status.
7. On failure, stop services, restore the previous binaries and, only if required, restore the pre-upgrade data backup.

Never downgrade or replace chain data across different Chain IDs or Genesis hashes. A Testnet database must not be opened as Devnet or Mainnet.

## Private/public Testnet boundary

This deployment material is packaging/service evidence only and does not by itself complete final validation. Under Master-TZ v0.2.12, final distributed validation is session-based: at least three independently launched nodes/clients, at least one real reachable bootstrap route, and at least three separate sessions of approximately 2-3 hours covering restart, bootstrap-loss/recovery, mining, transactions, peer exchange, reorg observation and installer/Desktop use where applicable. There is no mandatory 24-hour or seven-day continuous powered-on gate for v0.2.
