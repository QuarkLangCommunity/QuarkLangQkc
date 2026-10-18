#!/usr/bin/env bash
# Shared recording preconditions for the checked-in benchmark harnesses.
#
# A record is only comparable when the host was quiet while it was taken, so every harness that can
# write a record uses this one probe. It is deliberately a library, not a script: the tide and claims
# harnesses must agree on what "quiet" means or their numbers are not comparable.
set -euo pipefail

# load1 prints this platform's 1-minute load average.
load1() {
  if [ -r /proc/loadavg ]; then
    LC_ALL=C awk '{ print $1; exit }' /proc/loadavg
  elif sysctl -n vm.loadavg >/dev/null 2>&1; then
    sysctl -n vm.loadavg | tr -d '{}' | LC_ALL=C awk '{ print $1; exit }'
  else
    python3 -c 'import os; print("%.2f" % os.getloadavg()[0])' 2>/dev/null
  fi
}

# cpu_count prints the logical CPU count, or 0 when no probe knows it.
cpu_count() {
  getconf _NPROCESSORS_ONLN 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || nproc 2>/dev/null || echo 0
}

# load_below succeeds when the load is strictly below the limit.
load_below() {
  LC_ALL=C awk -v measured="$1" -v limit="$2" 'BEGIN { exit !(measured < limit) }'
}

# check_record_load refuses a recording run on a busy host and sets LOAD_META for the record.
# Usage: check_record_load <label> <record-mode 0|1> <max-load-per-logical-cpu>
check_record_load() {
  local label="$1" record="$2" per_cpu="$3"
  if [ "$record" -eq 0 ]; then return 0; fi
  local cpus limit load
  cpus="$(cpu_count)"
  [[ "$cpus" =~ ^[1-9][0-9]*$ ]] || { echo "$label: --record needs a positive logical CPU count" >&2; exit 2; }
  limit="$(LC_ALL=C awk -v cpus="$cpus" -v per="$per_cpu" 'BEGIN { printf "%.2f", cpus * per }')"
  load="$(load1)"
  [[ "$load" =~ ^[0-9]+([.][0-9]+)?$ ]] || { echo "$label: could not read the 1-minute load average for --record" >&2; exit 2; }
  if ! load_below "$load" "$limit"; then
    echo "$label: refusing --record: 1-minute load average $load is not below $limit ($per_cpu per logical CPU on $cpus logical CPUs); wait for a quiet host" >&2
    exit 2
  fi
  LOAD_META="1-minute load average $load before the run; precondition below $limit ($per_cpu per logical CPU on $cpus logical CPUs)"
}
