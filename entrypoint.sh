#!/bin/sh

# TODO: this should probably be started using systemd?
./mock-routing-server -httpapi-addr=127.0.0.1:9999 -httpapi-path=/ > /var/log/mock-routing-server.log 2>&1 &

./hydra-booster -metrics-addr=0.0.0.0:8888 -httpapi-addr=0.0.0.0:7779 -ui-theme=none
