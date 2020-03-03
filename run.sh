#!/usr/bin/env bash
set -e

docker build -t dogapp:latest .

docker run --rm -e FREQUENCY=60 -e SLACK_TOKEN=$SLACK_TOKEN -v $(pwd)/db:/usr/src/app/db -it dogapp:latest
