local filter="$1"
shift
jq -n "$@" "$filter" 2>&1
