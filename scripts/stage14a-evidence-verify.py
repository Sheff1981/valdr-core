#!/usr/bin/env python3
import hashlib
import json
import pathlib
import re
import sys

EXPECTED_NETWORK = "testnet2"
EXPECTED_CHAIN_ID = "valdr-testnet-2"
SESSION_SCHEMA = "valdr-stage14a-session-v1"
SUMMARY_SCHEMA = "valdr-stage14a-session-summary-v1"
COMMIT_RE = re.compile(r"^[0-9a-f]{40}$")
SAFE_ID_RE = re.compile(r"^[A-Za-z0-9._-]+$")


def fail(message):
    raise ValueError(message)


def load_json(path):
    return json.loads(path.read_text(encoding="utf-8"))


def parse_sums(path):
    result = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        if not raw.strip():
            continue
        try:
            digest, name = raw.split("  ", 1)
        except ValueError as exc:
            raise ValueError(f"{path}: invalid checksum line: {raw!r}") from exc
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            fail(f"{path}: invalid SHA-256 digest for {name}")
        if name in result:
            fail(f"{path}: duplicate checksum entry for {name}")
        result[name] = digest
    return result


def sha256_file(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def require_safe_id(value, label, session_dir):
    if not isinstance(value, str) or not SAFE_ID_RE.fullmatch(value):
        fail(f"{session_dir}: {label} must contain only A-Z a-z 0-9 . _ -")
    return value


def verify_session(session_dir):
    manifest_path = session_dir / "manifest.json"
    snapshots_path = session_dir / "snapshots.jsonl"
    summary_path = session_dir / "summary.json"
    events_path = session_dir / "events.jsonl"
    sums_path = session_dir / "SHA256SUMS"
    for path in (manifest_path, snapshots_path, summary_path, events_path, sums_path):
        if not path.is_file():
            fail(f"{session_dir}: missing {path.name}")

    sums = parse_sums(sums_path)
    expected_names = {"manifest.json", "snapshots.jsonl", "summary.json", "events.jsonl"}
    if set(sums) != expected_names:
        fail(f"{session_dir}: checksum set mismatch: {sorted(sums)}")
    for name, expected in sums.items():
        actual = sha256_file(session_dir / name)
        if actual != expected:
            fail(f"{session_dir}: checksum mismatch for {name}")

    manifest = load_json(manifest_path)
    summary = load_json(summary_path)

    if manifest.get("schema") != SESSION_SCHEMA:
        fail(f"{session_dir}: unexpected manifest schema")
    if summary.get("schema") != SUMMARY_SCHEMA:
        fail(f"{session_dir}: unexpected summary schema")
    if manifest.get("network") != EXPECTED_NETWORK or summary.get("network") != EXPECTED_NETWORK:
        fail(f"{session_dir}: wrong network")
    if manifest.get("chain_id") != EXPECTED_CHAIN_ID or summary.get("chain_id") != EXPECTED_CHAIN_ID:
        fail(f"{session_dir}: wrong chain id")

    session_id = require_safe_id(manifest.get("session_id"), "session_id", session_dir)
    group_id = require_safe_id(manifest.get("session_group_id"), "session_group_id", session_dir)
    if summary.get("session_id") != session_id:
        fail(f"{session_dir}: session id mismatch")
    if summary.get("session_group_id") != group_id:
        fail(f"{session_dir}: session group mismatch")

    commit = str(manifest.get("source_commit") or "").lower()
    if not COMMIT_RE.fullmatch(commit):
        fail(f"{session_dir}: source_commit must be an exact 40-character git SHA")
    if str(summary.get("source_commit") or "").lower() != commit:
        fail(f"{session_dir}: summary source_commit mismatch")

    machine_id = require_safe_id(manifest.get("machine_id"), "machine_id", session_dir)

    if manifest.get("acceptance_scope") != "local-session-evidence-only":
        fail(f"{session_dir}: unexpected acceptance scope")
    if summary.get("stage14a_pass") is not False:
        fail(f"{session_dir}: local session must never claim Stage 14A PASS")
    if summary.get("result") != "local_observation_complete":
        fail(f"{session_dir}: unexpected local result")

    count = summary.get("snapshot_count")
    if not isinstance(count, int) or count < 1:
        fail(f"{session_dir}: snapshot_count must be positive")
    start_height = int(summary.get("start_height"))
    end_height = int(summary.get("end_height"))
    if end_height < start_height:
        fail(f"{session_dir}: height regressed")
    start_work = int(str(summary.get("start_chainwork")), 16)
    end_work = int(str(summary.get("end_chainwork")), 16)
    if end_work < start_work:
        fail(f"{session_dir}: chainwork regressed")

    snapshots = 0
    tip_hashes = set()
    with snapshots_path.open("r", encoding="utf-8") as f:
        for line in f:
            if not line.strip():
                continue
            item = json.loads(line)
            status = item.get("status") or {}
            if status.get("network") != EXPECTED_NETWORK or status.get("chain_id") != EXPECTED_CHAIN_ID:
                fail(f"{session_dir}: snapshot network identity mismatch")
            tip_hash = status.get("tip_hash")
            if not isinstance(tip_hash, str) or not tip_hash:
                fail(f"{session_dir}: snapshot missing tip_hash")
            tip_hashes.add(tip_hash)
            snapshots += 1
    if snapshots != count:
        fail(f"{session_dir}: snapshot_count does not match snapshots.jsonl")

    event_types = set()
    with events_path.open("r", encoding="utf-8") as f:
        for line in f:
            if not line.strip():
                continue
            event = json.loads(line)
            if event.get("session_group_id") != group_id:
                fail(f"{session_dir}: event session group mismatch")
            if event.get("machine_id") != machine_id:
                fail(f"{session_dir}: event machine mismatch")
            event_type = event.get("event_type")
            if not isinstance(event_type, str) or not event_type:
                fail(f"{session_dir}: event_type is required")
            event_types.add(event_type)

    return {
        "session_id": session_id,
        "session_group_id": group_id,
        "machine_id": machine_id,
        "operator": manifest.get("operator"),
        "source_commit": commit,
        "bootstrap_route": manifest.get("bootstrap_route"),
        "planned_duration_seconds": manifest.get("planned_duration_seconds"),
        "snapshot_count": count,
        "start_height": start_height,
        "end_height": end_height,
        "_tip_hashes": tip_hashes,
        "_event_types": event_types,
    }


def discover(args):
    if not args:
        fail("usage: stage14a-evidence-verify.py SESSION_DIR [SESSION_DIR ...] | EVIDENCE_ROOT")
    paths = [pathlib.Path(a) for a in args]
    if len(paths) == 1 and paths[0].is_dir() and not (paths[0] / "manifest.json").exists():
        paths = sorted(p for p in paths[0].iterdir() if p.is_dir() and (p / "manifest.json").exists())
    if not paths:
        fail("no Stage 14A session directories found")
    return paths


def public_session(session):
    return {k: v for k, v in session.items() if not k.startswith("_")}


def main():
    try:
        dirs = discover(sys.argv[1:])
        evidence = [verify_session(p) for p in dirs]
        ids = [s["session_id"] for s in evidence]
        commits = sorted({s["source_commit"] for s in evidence})

        if len(set(ids)) != len(ids):
            fail("duplicate session_id in consolidated evidence")

        grouped = {}
        seen_group_machine = set()
        for item in evidence:
            pair = (item["session_group_id"], item["machine_id"])
            if pair in seen_group_machine:
                fail(
                    f"duplicate machine evidence for distributed session "
                    f"{item['session_group_id']}: {item['machine_id']}"
                )
            seen_group_machine.add(pair)
            grouped.setdefault(item["session_group_id"], []).append(item)

        group_results = []
        all_recorded_event_types = set()
        all_groups_have_three_machines = True
        all_groups_have_bootstrap = True
        all_groups_have_common_tip = True
        for group_id in sorted(grouped):
            items = grouped[group_id]
            machines = sorted({x["machine_id"] for x in items})
            routes = sorted({
                str(x["bootstrap_route"]).strip()
                for x in items
                if x.get("bootstrap_route") and str(x["bootstrap_route"]).strip()
            })
            common_tips = set(items[0]["_tip_hashes"])
            group_event_types = set()
            group_event_types.update(items[0]["_event_types"])
            for item in items[1:]:
                common_tips.intersection_update(item["_tip_hashes"])
                group_event_types.update(item["_event_types"])
            all_recorded_event_types.update(group_event_types)

            has_three = len(machines) >= 3
            has_bootstrap = bool(routes)
            has_common_tip = bool(common_tips)
            all_groups_have_three_machines &= has_three
            all_groups_have_bootstrap &= has_bootstrap
            all_groups_have_common_tip &= has_common_tip

            group_results.append({
                "session_group_id": group_id,
                "evidence_count": len(items),
                "machine_count": len(machines),
                "machines": machines,
                "bootstrap_routes": routes,
                "common_tip_observed": has_common_tip,
                "common_tip_hashes": sorted(common_tips),
                "operator_recorded_event_types": sorted(group_event_types),
            })

        checks = {
            "at_least_three_distributed_sessions": len(group_results) >= 3,
            "each_session_has_at_least_three_independent_machine_labels": all_groups_have_three_machines,
            "single_exact_source_commit": len(commits) == 1,
            "bootstrap_route_recorded_each_session": all_groups_have_bootstrap,
            "common_tip_observed_each_session": all_groups_have_common_tip,
            "all_integrity_checks_passed": True,
        }
        ready = all(checks.values())
        machines = sorted({s["machine_id"] for s in evidence})
        result = {
            "schema": "valdr-stage14a-consolidated-check-v2",
            "network": EXPECTED_NETWORK,
            "chain_id": EXPECTED_CHAIN_ID,
            "session_count": len(group_results),
            "evidence_count": len(evidence),
            "machine_count": len(machines),
            "machines": machines,
            "source_commits": commits,
            "checks": checks,
            "distributed_sessions": group_results,
            "operator_recorded_scenario_coverage": sorted(all_recorded_event_types),
            "automated_evidence_ready": ready,
            "stage14a_pass": False,
            "human_review_required": True,
            "duration_and_scenario_acceptance_checked": False,
            "note": (
                "Automated evidence readiness is not Stage 14A acceptance. "
                "Human review must confirm that machine labels represent genuinely independent clients, "
                "bootstrap routes were really reachable, session duration was approximately 2-3 hours, "
                "required mining/transaction/restart/recovery scenarios were exercised, and no unresolved "
                "consensus or critical blockers remain."
            ),
            "evidence": [public_session(s) for s in evidence],
        }
        print(json.dumps(result, indent=2, sort_keys=True))
        return 0 if ready else 1
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"stage14a evidence verification failed: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
