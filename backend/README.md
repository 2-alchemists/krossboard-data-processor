# Installation

* Set user and root installation directory

  ```
  $ export KOAMC_USER=koamc
  $ export KOAMC_ROOT_DIR=/opt/$KOAMC_USER
  ```

* Create koamc user

  ```
  $ sudo useradd $KOAMC_USER -m --home-dir $KOAMC_ROOT_DIR
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
  $ sudo install -m 644 ./scripts/kube-opex-analytics-mc.service.env $KOAMC_ROOT_DIR/etc/
  $ sudo install -m 644 ./scripts/kube-opex-analytics-mc.service /lib/systemd/system/
  ```  

* Fix permissions on directories

  ```
  $ sudo chown -R $KOAMC_USER:$KOAMC_USER \
        $KOAMC_ROOT_DIR/{data,run}
  ```  