local url="$1" header
shift
set -- "$@" --
while [ "$1" != "--" ]; do
  header=$1
  shift
  set -- "$@" -H "$header"
done
shift
curl --fail --silent --show-error "$url" "$@" 2>&1
