local input="$1" path="$2" value="$3" filter
filter=$path' += [$value]'
GOTOSH_RT_jqjson__Filter "$input" "$filter" --argjson value "$value"
