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


def verify_session(session_dir):
    manifest_path = session_dir / "manifest.json"
    snapshots_path = session_dir / "snapshots.jsonl"
    summary_path = session_dir / "summary.json"
    sums_path = session_dir / "SHA256SUMS"
    for path in (manifest_path, snapshots_path, summary_path, sums_path):
        if not path.is_file():
            fail(f"{session_dir}: missing {path.name}")

    sums = parse_sums(sums_path)
    expected_names = {"manifest.json", "snapshots.jsonl", "summary.json"}
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

    session_id = manifest.get("session_id")
    if not session_id or summary.get("session_id") != session_id:
        fail(f"{session_dir}: session id mismatch")
    commit = str(manifest.get("source_commit") or "").lower()
    if not COMMIT_RE.fullmatch(commit):
        fail(f"{session_dir}: source_commit must be an exact 40-character git SHA")
    if str(summary.get("source_commit") or "").lower() != commit:
        fail(f"{session_dir}: summary source_commit mismatch")

    machine_id = manifest.get("machine_id")
    if not isinstance(machine_id, str) or not machine_id.strip():
        fail(f"{session_dir}: machine_id is required for consolidated Stage 14A evidence")

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
    with snapshots_path.open("r", encoding="utf-8") as f:
        for line in f:
            if not line.strip():
                continue
            item = json.loads(line)
            status = item.get("status") or {}
            if status.get("network") != EXPECTED_NETWORK or status.get("chain_id") != EXPECTED_CHAIN_ID:
                fail(f"{session_dir}: snapshot network identity mismatch")
            snapshots += 1
    if snapshots != count:
        fail(f"{session_dir}: snapshot_count does not match snapshots.jsonl")

    return {
        "session_id": session_id,
        "machine_id": machine_id.strip(),
        "operator": manifest.get("operator"),
        "source_commit": commit,
        "bootstrap_route": manifest.get("bootstrap_route"),
        "planned_duration_seconds": manifest.get("planned_duration_seconds"),
        "snapshot_count": count,
        "start_height": start_height,
        "end_height": end_height,
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


def main():
    try:
        dirs = discover(sys.argv[1:])
        sessions = [verify_session(p) for p in dirs]
        ids = [s["session_id"] for s in sessions]
        machines = [s["machine_id"] for s in sessions]
        commits = sorted({s["source_commit"] for s in sessions})
        bootstrap_routes = sorted({str(s["bootstrap_route"]).strip() for s in sessions if s["bootstrap_route"]})

        if len(set(ids)) != len(ids):
            fail("duplicate session_id in consolidated evidence")

        checks = {
            "at_least_three_sessions": len(sessions) >= 3,
            "at_least_three_independent_machine_labels": len(set(machines)) >= 3,
            "single_exact_source_commit": len(commits) == 1,
            "bootstrap_route_recorded": bool(bootstrap_routes),
            "all_integrity_checks_passed": True,
        }
        ready = all(checks.values())
        result = {
            "schema": "valdr-stage14a-consolidated-check-v1",
            "network": EXPECTED_NETWORK,
            "chain_id": EXPECTED_CHAIN_ID,
            "session_count": len(sessions),
            "machine_count": len(set(machines)),
            "source_commits": commits,
            "bootstrap_routes": bootstrap_routes,
            "checks": checks,
            "automated_evidence_ready": ready,
            "stage14a_pass": False,
            "human_review_required": True,
            "duration_and_scenario_acceptance_checked": False,
            "note": (
                "Automated evidence readiness is not Stage 14A acceptance. "
                "Human review must confirm independent machines, real bootstrap reachability, "
                "2-3 hour session intent, mining/transaction/restart/recovery scenarios, "
                "and absence of unresolved consensus or critical blockers."
            ),
            "sessions": sessions,
        }
        print(json.dumps(result, indent=2, sort_keys=True))
        return 0 if ready else 1
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"stage14a evidence verification failed: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
