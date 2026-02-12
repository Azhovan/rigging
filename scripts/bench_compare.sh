#!/usr/bin/env bash
set -euo pipefail

BASE_REF="${1:-${BASE_REF:-main}}"
BENCH_RE="${2:-${BENCH_RE:-BenchmarkCreateSnapshot_SmallConfig$}}"
COUNT="${3:-${BENCH_COUNT:-5}}"
PKG="${4:-${BENCH_PKG:-.}}"

ROOT="$(git rev-parse --show-toplevel)"
BENCHSTAT_BIN=""

if command -v benchstat >/dev/null 2>&1; then
	BENCHSTAT_BIN="benchstat"
else
	GOPATH_BIN="$(go env GOPATH)/bin/benchstat"
	if [ -x "${GOPATH_BIN}" ]; then
		BENCHSTAT_BIN="${GOPATH_BIN}"
	fi
fi

if [ -z "${BENCHSTAT_BIN}" ]; then
	echo "benchstat not found."
	echo "Install with: go install golang.org/x/perf/cmd/benchstat@latest"
	exit 1
fi

if ! git -C "${ROOT}" rev-parse --verify "${BASE_REF}^{commit}" >/dev/null 2>&1; then
	echo "Base ref '${BASE_REF}' not found."
	exit 1
fi

WORKTREE_DIR="$(mktemp -d /tmp/rigging-bench-worktree-XXXXXX)"
BASE_OUT="$(mktemp /tmp/rigging-bench-base-XXXXXX)"
HEAD_OUT="$(mktemp /tmp/rigging-bench-head-XXXXXX)"

cleanup() {
	if git -C "${ROOT}" worktree list --porcelain | grep -F "worktree ${WORKTREE_DIR}" >/dev/null 2>&1; then
		git -C "${ROOT}" worktree remove --force "${WORKTREE_DIR}" >/dev/null 2>&1 || true
	fi
	rm -rf "${WORKTREE_DIR}"
	rm -f "${BASE_OUT}" "${HEAD_OUT}"
}
trap cleanup EXIT

require_benchmark_results() {
	local file="$1"
	local label="$2"

	if ! grep -q '^Benchmark' "${file}"; then
		echo "No benchmark results found for ${label}."
		echo "Ensure BENCH/BENCH_RE matches benchmarks present in that ref."
		exit 1
	fi
}

echo "Benchmark config:"
echo "  base_ref=${BASE_REF}"
echo "  bench=${BENCH_RE}"
echo "  count=${COUNT}"
echo "  pkg=${PKG}"
echo ""

echo "Running baseline benchmark (${BASE_REF})..."
git -C "${ROOT}" worktree add --detach "${WORKTREE_DIR}" "${BASE_REF}" >/dev/null 2>&1
(
	cd "${WORKTREE_DIR}"
	GOCACHE="${GOCACHE:-/tmp/rigging-gocache}" GOTMPDIR="${GOTMPDIR:-/tmp}" \
		go test "${PKG}" -run '^$' -bench "${BENCH_RE}" -benchmem -count "${COUNT}" > "${BASE_OUT}"
)
require_benchmark_results "${BASE_OUT}" "base ref (${BASE_REF})"

echo "Running current benchmark (working tree)..."
(
	cd "${ROOT}"
	GOCACHE="${GOCACHE:-/tmp/rigging-gocache}" GOTMPDIR="${GOTMPDIR:-/tmp}" \
		go test "${PKG}" -run '^$' -bench "${BENCH_RE}" -benchmem -count "${COUNT}" > "${HEAD_OUT}"
)
require_benchmark_results "${HEAD_OUT}" "current tree"

echo ""
echo "benchstat (${BASE_REF} -> current):"
"${BENCHSTAT_BIN}" "${BASE_OUT}" "${HEAD_OUT}"
