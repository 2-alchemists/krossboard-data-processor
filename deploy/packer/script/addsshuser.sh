#!/bin/bash -eu

date > /etc/box_build_time

SSH_USER=${SSH_USERNAME:-ubuntu}
SSH_PASS=${SSH_PASSWORD:-ubuntu}
SSH_USER_HOME=${SSH_USER_HOME:-/home/${SSH_USER}}

# Create ubuntu user (if not already present)
if ! id -u $SSH_USER >/dev/null 2>&1; then
    echo "==> Creating $SSH_USER user"
    /usr/sbin/groupadd $SSH_USER
    /usr/sbin/useradd $SSH_USER -g $SSH_USER -G sudo -d $SSH_USER_HOME --create-home
    echo "${SSH_USER}:${SSH_PASS}" | chpasswd
fi

# Set up sudo
echo "==> Giving ${SSH_USER} sudo powers"
echo "${SSH_USER}        ALL=(ALL)       NOPASSWD: ALL" >> /etc/sudoers.d/$SSH_USER
chmod 440 /etc/sudoers.d/$SSH_USER

# Fix stdin not being a tty
if grep -q -E "^mesg n$" /root/.profile && sed -i "s/^mesg n$/tty -s \\&\\& mesg n/g" /root/.profile; then
    echo "==> Fixed stdin not being a tty."
fi

echo "==> Installing ubuntu key"
mkdir $SSH_USER_HOME/.ssh
chmod 700 $SSH_USER_HOME/.ssh
cd $SSH_USER_HOME/.ssh
