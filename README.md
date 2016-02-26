# telegraf-lxc-stats
LXC stats telegraf plugin

1.
Put your telegraf-lxc-stats binary to /usr/local/sbin/telegraf-lxc-stats

2.
Add to telegraf configuration file (e.g. /etc/telegraf.telegraf.conf):
```
[[inputs.exec]]
  command = "sudo /usr/local/sbin/telegraf-lxc-stats"
  data_format = "influx"
```
3.
Add to /etc/sudoers file (telegraf daemon is running by telegraf user by default):
```
telegraf ALL = NOPASSWD: /usr/local/sbin/telegraf-lxc-stats
```
