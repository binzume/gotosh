local input="$1" filter="$2"
shift 2
printf '%s' "$input" | jq "$@" "$filter" 2>&1
