#!/bin/bash
#ApplicationStart
PROCESS_NUM=$(ps -ef | grep "svscanboot" | grep -v "grep" | wc -l)
if [ $PROCESS_NUM -eq 0 ]; then
    echo "starting svscan!!!"
    sudo systemctl start svscan.service &
    sleep 5
fi
sudo svc -u /etc/service/relay
