local s="$1" needle="$2" i n
i=0
n=${#needle}
while [ "$i" -le "$(( ${#s} - n ))" ]; do
  if [ "${s:i:n}" = "$needle" ]; then
    printf '%d\n' "$i"
    return 0
  fi
  i=$((i + 1))
done
printf '%d\n' -1
