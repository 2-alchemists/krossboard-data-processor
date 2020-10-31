#!/bin/bash

##############################################################
#                                                            #
# This script allows to change the password of Krossboard UI #
# Copyrights 2020 2Alchemists - No copy is allowed.          #
#                                                            #
##############################################################

set -u
set -e

RED_COLOR='\033[0;31m'
NO_COLOR='\033[0m'

source /opt/krossboard/etc/krossboard.env

CADDYFILE=/opt/krossboard/etc/Caddyfile

MIN_PASSWORD_LENGTH=6

echo -n "  Type the old password:" 
read -s in_old_pass
echo

echo -n "  Type the new password (min $MIN_PASSWORD_LENGTH characters):" 
read -s in_new_pass
echo

echo -n "  Confirm new password:" 
read -s in_confirm_pass
echo

old_pass_matched=$(curl -o /dev/null -sSf -ukrossboard:$in_old_pass http://127.0.0.1/ || echo $?)
if [ "$old_pass_matched" != "" ]; then
    echo "=> [ERROR] could not validate the old password"
    exit 1
fi

if [ ${#in_new_pass} -lt ${MIN_PASSWORD_LENGTH} ]; then
    echo "=> [ERROR] the password must have at least ${MIN_PASSWORD_LENGTH} characters"
    exit 1
fi

if [ "$in_new_pass" != "$in_confirm_pass" ]; then
    echo "=> [ERROR] new password and confirmation do not match"
    exit 1
fi

new_password_hashed=$(docker run --rm ${KROSSBOARD_UI_IMAGE} caddy hash-password --plaintext "$in_new_pass")
echo "$new_password_hashed"

sed -i -E 's/(krossboard\s)[[:alnum:]]+/\1'$new_password_hashed'/' ${CADDYFILE}

echo "=> [SUCCESS] Password changed!"

echo -e "${RED_COLOR}You must restart the service to make the change effective => sudo systemctl restart krossboard-ui${NO_COLOR}"
