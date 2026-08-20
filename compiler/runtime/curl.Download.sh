local url="$1" path="$2" header status
shift 2
set -- "$@" --
while [ "$1" != "--" ]; do
  header=$1
  shift
  set -- "$@" -H "$header"
done
shift
curl --fail --silent --show-error --output "$path" "$url" "$@" 2>&1
status=$?
printf '%d\n' "$status"
return "$status"
