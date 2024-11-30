#!/bin/sh

echo "============ Go ============"
go run examples/$1.go | tee $1.out_go
echo "============ Bash ============"
go run . examples/$1.go > $1.sh
bash $1.sh 2>&1 | tee $1.out_sh
echo "============ Diff ============"
diff $1.out_go $1.out_sh
echo "============ $1 finished ============"
