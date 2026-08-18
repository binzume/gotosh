local s="$1" chars="$2" ch i
i=0
while [ "$i" -lt "${#s}" ]; do
  ch=${s:i:1}
  case "$chars" in
    *"$ch"*) printf '%d\n' "$i"; return 0 ;;
  esac
  i=$((i + 1))
done
printf '%d\n' -1
