#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

go test ./wallet \
  -run '^(TestEncryptedWalletBackupAndRestore|TestImportEncryptedVerifiedRequiresCorrectPassphraseBeforeWrite|TestImportEncryptedVerifiedTamperDoesNotWriteDestination)$' \
  -count=1

printf 'VALDR encrypted wallet recovery QA: PASS\n'
