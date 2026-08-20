local method="$1" url="$2" body="$3" header
shift 3
set -- "$@" --
while [ "$1" != "--" ]; do
  header=$1
  shift
  set -- "$@" -H "$header"
done
shift
curl --fail --silent --show-error --request "$method" --data "$body" "$url" "$@" 2>&1
