#!/bin/bash
PROCESS_NUM=$(ps -ef | grep "svscanboot" | grep -v "grep" | wc -l)
if [ $PROCESS_NUM -eq 0 ]; then
#check svc exists
    which svc
    if [[ $? != 0 ]]; then
		echo "daemontools no intalled, begin install!!!"
       	apt-get update
       	apt install daemontools
    fi
	if [ ! -f /lib/systemd/system/svscan.service ]; then
		echo "svscan job not config, begin install!!!"
	    echo $'[Unit]\nDescription=svscan job\n\n[Service]\nType=forking\nExecStart=/usr/bin/svscanboot\nRestart=on-failure\nTimeoutSec=infinity\n\n[Install]\nWantedBy=multi-user.target' > /lib/systemd/system/svscan.service
		systemctl enable svscan.service
		systemctl daemon-reload
	fi
fi
