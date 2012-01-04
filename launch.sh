#!/bin/sh

PIDS=""

cleanup() {
	for i in $PIDS; do
		echo "Killing $i"
		kill $i
	done
}

trap cleanup 1 2 3 6

NUM=$1
shift 1

for i in `seq 1 $NUM`; do
	(./tdm -station=$i $@) &

	if [ -z "$PIDS" ]; then
		PIDS="$!"
	else
		PIDS="$PIDS $!"
	fi
done

for i in $PIDS; do
	echo "Waiting for $i"
	wait $i
done


