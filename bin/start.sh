#!/bin/bash
#ApplicationStart

PROCESS_NUM=$(ps -ef | grep "svscanboot" | grep -v "grep" | wc -l)
if [ $PROCESS_NUM -eq 0 ]; then
    echo "starting svscan!!!"
    sudo systemctl start svscan.service &
    sleep 5
fi

LOG_DIR=/var/log/relay
if [ ! -d $LOG_DIR ]; then
    sudo mkdir -p $LOG_DIR
    sudo chown -R ubuntu:ubuntu $LOG_DIR
fi

sudo svc -u /etc/service/relay
