local url="$1" body="$2" header
shift 2
set -- "$@" --
while [ "$1" != "--" ]; do
  header=$1
  shift
  set -- "$@" -H "$header"
done
shift
curl --fail --silent --show-error --request POST --data "$body" "$url" "$@" 2>&1
