#!/usr/bin/env python3
import argparse
import datetime as dt
import hashlib
import json
import pathlib
import re
import sys

ALLOWED = {
    "automatic_join",
    "peer_handshake",
    "peer_exchange",
    "bootstrap_loss",
    "restart",
    "peer_cache_reconnect",
    "db_verify",
    "mining",
    "transaction",
    "pending_relay",
    "block_propagation",
    "installer_upgrade",
    "mempool",
    "desktop_sync",
    "package_start",
    "wallet_backup_restore",
    "reorg_observation",
    "crash_error",
    "note",
}

def sha256_file(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()

def rewrite_sums(session_dir):
    names = ["manifest.json", "snapshots.jsonl", "summary.json", "events.jsonl"]
    if not all((session_dir / name).is_file() for name in names):
        return
    lines = [f"{sha256_file(session_dir / name)}  {name}" for name in names]
    (session_dir / "SHA256SUMS").write_text("\n".join(lines) + "\n", encoding="utf-8")

def main():
    parser = argparse.ArgumentParser(description="Append one operator-recorded Stage 14A event.")
    parser.add_argument("session_dir")
    parser.add_argument("event_type", choices=sorted(ALLOWED))
    parser.add_argument("--result", choices=["pass", "fail", "observed", "not_applicable"], default="observed")
    parser.add_argument("--note", default="")
    args = parser.parse_args()

    session_dir = pathlib.Path(args.session_dir)
    manifest_path = session_dir / "manifest.json"
    events_path = session_dir / "events.jsonl"
    if not manifest_path.is_file():
        print(f"missing manifest: {manifest_path}", file=sys.stderr)
        return 2

    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    group_id = manifest.get("session_group_id")
    machine_id = manifest.get("machine_id")
    if not isinstance(group_id, str) or not re.fullmatch(r"[A-Za-z0-9._-]+", group_id):
        print("manifest has invalid session_group_id", file=sys.stderr)
        return 2
    if not isinstance(machine_id, str) or not re.fullmatch(r"[A-Za-z0-9._-]+", machine_id):
        print("manifest has invalid machine_id", file=sys.stderr)
        return 2

    event = {
        "schema": "valdr-stage14a-event-v1",
        "recorded_at": dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z"),
        "session_group_id": group_id,
        "machine_id": machine_id,
        "event_type": args.event_type,
        "result": args.result,
        "note": args.note,
        "evidence_kind": "operator_recorded",
    }
    with events_path.open("a", encoding="utf-8") as f:
        f.write(json.dumps(event, sort_keys=True) + "\n")
    rewrite_sums(session_dir)
    print(json.dumps(event, sort_keys=True))
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
