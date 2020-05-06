#!/bin/bash

##############################################################
#                                                            #
# This script allows to change the password of Krossboard UI #
# Copyrights 2020 2Alchemists - No copy is allowed.          #
#                                                            #
##############################################################

set -u
set -e

CADDYFILE=/opt/krossboard/etc/Caddyfile

MIN_PASSWORD_LENGTH=6

echo -n "Old password:" 
read -s in_old_pass
echo

echo -n "New password (min $MIN_PASSWORD_LENGTH characters):" 
read -s in_new_pass
echo

echo -n "Confirm new password:" 
read -s in_confirm_pass
echo

old_pass=$(grep -r 'basicauth' ${CADDYFILE} | cut -d' ' -f3)
if [ "$old_pass" != "$in_old_pass" ]; then
    echo "old password does not match"
    exit 1
fi

if [ ${#in_new_pass} -lt ${MIN_PASSWORD_LENGTH} ]; then
    echo "${#in_new_pass} ${in_new_pass}"
    echo "the password must have at least ${MIN_PASSWORD_LENGTH} characters"
    exit 1
fi

if [ "$in_new_pass" != "$in_confirm_pass" ]; then
    echo "new password and confirmation do not match"
    exit 1
fi

sed -i -E 's/(basicauth\s[[:alnum:]]+\s)[[:print:]]+(\s)/\1'$in_new_pass'\2/' ${CADDYFILE}

echo 'password changed'
