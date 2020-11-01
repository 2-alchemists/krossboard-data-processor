#!/bin/bash -eu
MOTD_FILE=/etc/motd
BANNER_WIDTH=64
PLATFORM_RELEASE=$(lsb_release -sd)
PLATFORM_MSG=$(printf '%s' "$PLATFORM_RELEASE")
printf '%0.1s' "-"{1..64} > ${MOTD_FILE}
printf '\n' >> ${MOTD_FILE}
printf '%2s%-20s%30s\n' " " "Krossboard:" "https://krossboard.app/" >> ${MOTD_FILE}
printf '%2s%-20s%30s\n' " " "Documentation:" "https://krossboard.app/docs/" >> ${MOTD_FILE}
printf '%2s%-20s%30s\n' " " "Issue Tracker:" "https://github.com/2-alchemists/krossboard" >> ${MOTD_FILE}
printf '%2s%-20s%30s\n' " " "Support:" "https://krossboard.app/contact/support/" >> ${MOTD_FILE}
printf '%0.1s' "-"{1..64} >> ${MOTD_FILE}
printf '\n' >> ${MOTD_FILE}
