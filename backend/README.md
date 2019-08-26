# Installation

* Set root installation directory

  ```
  $ export KOAMC_ROOT_DIR=/opt/koamc
  ```

* Create koamc user

  ```
  $ sudo useradd koamc -m --home-dir $KOAMC_ROOT_DIR
  ```

* Create installation tree

  ```
  $ sudo install -d $KOAMC_ROOT_DIR/{bin,data,etc,run}
  ```

  * Copy binaries

  ```
  $ sudo install -m 755 /path/to/binary $KOAMC_ROOT_DIR/bin/
  ```

* Copy systemd scripts

  ```
  $ sudo install 644 ./scripts/kube-opex-analytics-mc.service.env $KOAMC_ROOT_DIR/etc/
  $ sudo install 644 ./scripts/kube-opex-analytics-mc.service /lib/systemd/system/
  ```  