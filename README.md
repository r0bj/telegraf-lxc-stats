# telegraf-lxc-stats
LXC stats telegraf plugin

1.
Put telegraf-lxc-stats binary to e.g. /usr/local/sbin/telegraf-lxc-stats

2.
Add to telegraf configuration file (e.g. /etc/telegraf.telegraf.conf):
```
[[inputs.exec]]
  command = "sudo /usr/local/sbin/telegraf-lxc-stats"
  data_format = "influx"
```
3.
Add to /etc/sudoers file (telegraf daemon user by default):
```
telegraf ALL = NOPASSWD: /usr/local/sbin/telegraf-lxc-stats
```
