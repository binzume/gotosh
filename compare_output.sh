#!/bin/sh
IN=go.mod
OUT_DIR=./out
PARAMS='aa bb 123 456'

mkdir -p $OUT_DIR
go run . examples/$1.go > $OUT_DIR/$1.sh
go run examples/$1.go $PARAMS < $IN > $OUT_DIR/$1.out_go
echo "============ $1.sh ============"
bash $OUT_DIR/$1.sh $PARAMS 2>&1 < $IN | tee $OUT_DIR/$1.out_sh
echo "============ Diff ============"
diff $OUT_DIR/$1.out_go $OUT_DIR/$1.out_sh
ret=$?
echo "============ $1 finished ============"
exit $ret
