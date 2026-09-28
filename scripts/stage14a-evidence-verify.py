#!/usr/bin/env python3
import datetime as dt
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
EVENT_TYPES = {
    "peer_exchange",
    "bootstrap_loss",
    "restart",
    "db_verify",
    "mining",
    "transaction",
    "mempool",
    "desktop_sync",
    "package_start",
    "wallet_backup_restore",
    "reorg_observation",
    "crash_error",
    "note",
}
EVENT_RESULTS = {"pass", "fail", "observed", "not_applicable"}
UTC_RE = re.compile(r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$")


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
    success_count = summary.get("successful_snapshot_count", count)
    error_count = summary.get("observation_error_count", 0)
    if not isinstance(count, int) or count < 1:
        fail(f"{session_dir}: snapshot_count must be positive")
    if not isinstance(success_count, int) or success_count < 1:
        fail(f"{session_dir}: successful_snapshot_count must be positive")
    if not isinstance(error_count, int) or error_count < 0:
        fail(f"{session_dir}: observation_error_count must be non-negative")
    if success_count + error_count != count:
        fail(f"{session_dir}: snapshot summary counts are inconsistent")
    start_height = int(summary.get("start_height"))
    end_height = int(summary.get("end_height"))
    if end_height < start_height:
        fail(f"{session_dir}: height regressed")
    start_work = int(str(summary.get("start_chainwork")), 16)
    end_work = int(str(summary.get("end_chainwork")), 16)
    if end_work < start_work:
        fail(f"{session_dir}: chainwork regressed")

    snapshots = 0
    successful_snapshots = 0
    observed_errors = 0
    tip_hashes = set()
    successful_statuses = []
    with snapshots_path.open("r", encoding="utf-8") as f:
        for line in f:
            if not line.strip():
                continue
            item = json.loads(line)
            snapshots += 1
            if "observation_error" in item:
                err = item.get("observation_error")
                if not isinstance(err, dict) or not err.get("component"):
                    fail(f"{session_dir}: malformed observation_error snapshot")
                observed_errors += 1
                continue
            status = item.get("status") or {}
            if status.get("network") != EXPECTED_NETWORK or status.get("chain_id") != EXPECTED_CHAIN_ID:
                fail(f"{session_dir}: snapshot network identity mismatch")
            tip_hash = status.get("tip_hash")
            if not isinstance(tip_hash, str) or not tip_hash:
                fail(f"{session_dir}: snapshot missing tip_hash")
            tip_hashes.add(tip_hash)
            successful_statuses.append(status)
            successful_snapshots += 1
    if snapshots != count:
        fail(f"{session_dir}: snapshot_count does not match snapshots.jsonl")
    if successful_snapshots != success_count:
        fail(f"{session_dir}: successful_snapshot_count does not match snapshots.jsonl")
    if observed_errors != error_count:
        fail(f"{session_dir}: observation_error_count does not match snapshots.jsonl")

    first_status = successful_statuses[0]
    last_status = successful_statuses[-1]
    if int(first_status.get("height")) != start_height:
        fail(f"{session_dir}: summary start_height does not match first successful snapshot")
    if int(last_status.get("height")) != end_height:
        fail(f"{session_dir}: summary end_height does not match last successful snapshot")
    if summary.get("start_tip_hash") != first_status.get("tip_hash"):
        fail(f"{session_dir}: summary start_tip_hash does not match first successful snapshot")
    if summary.get("end_tip_hash") != last_status.get("tip_hash"):
        fail(f"{session_dir}: summary end_tip_hash does not match last successful snapshot")
    if int(str(summary.get("start_chainwork")), 16) != int(str(first_status.get("chainwork")), 16):
        fail(f"{session_dir}: summary start_chainwork does not match first successful snapshot")
    if int(str(summary.get("end_chainwork")), 16) != int(str(last_status.get("chainwork")), 16):
        fail(f"{session_dir}: summary end_chainwork does not match last successful snapshot")

    manifest_started = manifest.get("started_at")
    summary_started = summary.get("started_at")
    summary_ended = summary.get("ended_at")
    for label, value in (
        ("manifest started_at", manifest_started),
        ("summary started_at", summary_started),
        ("summary ended_at", summary_ended),
    ):
        if not isinstance(value, str) or not UTC_RE.fullmatch(value):
            fail(f"{session_dir}: invalid {label}")
    if manifest_started != summary_started:
        fail(f"{session_dir}: manifest/summary started_at mismatch")
    started_dt = dt.datetime.strptime(summary_started, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=dt.timezone.utc)
    ended_dt = dt.datetime.strptime(summary_ended, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=dt.timezone.utc)
    if ended_dt < started_dt:
        fail(f"{session_dir}: ended_at precedes started_at")
    observed_duration_seconds = int((ended_dt - started_dt).total_seconds())

    planned_duration_seconds = manifest.get("planned_duration_seconds")
    if not isinstance(planned_duration_seconds, int) or isinstance(planned_duration_seconds, bool) or planned_duration_seconds < 0:
        fail(f"{session_dir}: planned_duration_seconds must be a non-negative integer")

    event_types = set()
    failed_event_count = 0
    with events_path.open("r", encoding="utf-8") as f:
        for line in f:
            if not line.strip():
                continue
            event = json.loads(line)
            if event.get("schema") != "valdr-stage14a-event-v1":
                fail(f"{session_dir}: unexpected event schema")
            if event.get("session_group_id") != group_id:
                fail(f"{session_dir}: event session group mismatch")
            if event.get("machine_id") != machine_id:
                fail(f"{session_dir}: event machine mismatch")
            event_type = event.get("event_type")
            if event_type not in EVENT_TYPES:
                fail(f"{session_dir}: unsupported event_type: {event_type!r}")
            if event.get("result") not in EVENT_RESULTS:
                fail(f"{session_dir}: unsupported event result")
            if event.get("result") == "fail":
                failed_event_count += 1
            if event.get("evidence_kind") != "operator_recorded":
                fail(f"{session_dir}: unexpected event evidence_kind")
            recorded_at = event.get("recorded_at")
            if not isinstance(recorded_at, str) or not UTC_RE.fullmatch(recorded_at):
                fail(f"{session_dir}: invalid event recorded_at")
            event_types.add(event_type)

    return {
        "session_id": session_id,
        "session_group_id": group_id,
        "machine_id": machine_id,
        "operator": manifest.get("operator"),
        "source_commit": commit,
        "bootstrap_route": manifest.get("bootstrap_route"),
        "planned_duration_seconds": planned_duration_seconds,
        "observed_duration_seconds": observed_duration_seconds,
        "snapshot_count": count,
        "successful_snapshot_count": success_count,
        "observation_error_count": error_count,
        "start_height": start_height,
        "end_height": end_height,
        "_tip_hashes": tip_hashes,
        "_event_types": event_types,
        "_failed_event_count": failed_event_count,
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
        no_operator_recorded_failures = True
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
            group_failed_events = 0
            group_event_types.update(items[0]["_event_types"])
            group_failed_events += items[0]["_failed_event_count"]
            for item in items[1:]:
                common_tips.intersection_update(item["_tip_hashes"])
                group_event_types.update(item["_event_types"])
                group_failed_events += item["_failed_event_count"]
            all_recorded_event_types.update(group_event_types)
            if group_failed_events:
                no_operator_recorded_failures = False

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
                "operator_recorded_failed_event_count": group_failed_events,
            })

        checks = {
            "at_least_three_distributed_sessions": len(group_results) >= 3,
            "each_session_has_at_least_three_independent_machine_labels": all_groups_have_three_machines,
            "single_exact_source_commit": len(commits) == 1,
            "bootstrap_route_recorded_each_session": all_groups_have_bootstrap,
            "common_tip_observed_each_session": all_groups_have_common_tip,
            "no_operator_recorded_failures": no_operator_recorded_failures,
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
