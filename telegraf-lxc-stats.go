package main

import (
	"fmt"
	"strconv"
	"strings"
	"reflect"
	"regexp"
	"errors"
	"io/ioutil"

	"gopkg.in/lxc/go-lxc.v2"
	"github.com/shirou/gopsutil/mem"
)

func str_to_uint64(s string) uint64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		msg, err := fmt.Printf("Func str_to_uint64 fail for %s\n", s)
		if err != nil {
			panic("Func str_to_uint64 fail")
		} else {
			panic(msg)
		}
	}
	return uint64(i)
}

func gen_line_prot_msg(m map[string]map[string]interface{}) string {
	output_list := make([]string, 0)
	for lxc_host, lxc_data := range m {
		lxc_data_array := make([]string, 0)

		for key, value := range lxc_data {
			if t := reflect.TypeOf(value); t.Kind() == reflect.Uint64 {
				lxc_data_array = append(lxc_data_array, fmt.Sprintf("%s=%d", key, value))
			} else if t := reflect.TypeOf(value); t.Kind() == reflect.Float64 {
				lxc_data_array = append(lxc_data_array, fmt.Sprintf("%s=%f", key, value))
			}
		}
		line := "lxcstats,lxc_host=" + lxc_host + " " + strings.Join(lxc_data_array, ",")
		output_list = append(output_list, line)
	}
	return strings.Join(output_list, "\n")
}

// takes string from cgroup cpuset.cpus and return number of cores, eg. takes "0-3,26" and return 5 
func count_cores(cpus string) int {
	cpus_array := strings.Split(cpus, ",")
	r := regexp.MustCompile(`^(\d+)-(\d+)$`)
	var cntr int
	for _, entry := range cpus_array {
		if matches := r.FindStringSubmatch(entry); matches != nil {
			start_int := str_to_uint64(matches[1])
			stop_int := str_to_uint64(matches[2])
			for i := start_int; i <= stop_int; i++ {
				cntr++
			}
		} else if matches, _ := regexp.MatchString(`\d+`, entry); matches {
			cntr++
		}
	}
	return cntr
}

func blkio_serviced(c *lxc.Container) (map[string]uint64, error) {
	var read uint64 = 0
	var write uint64 = 0
	for _, v := range c.CgroupItem("blkio.throttle.io_serviced") {
		b := strings.Split(v, " ")
		if b[1] == "Read" {
			read += str_to_uint64(b[2])
		}
		if b[1] == "Write" {
			write += str_to_uint64(b[2])
		}
	}
	return map[string]uint64{"blkio_serviced_read": read, "blkio_serviced_write": write}, nil
}

func blkio_service_bytes(c *lxc.Container) (map[string]uint64, error) {
	var read uint64 = 0
	var write uint64 = 0
	for _, v := range c.CgroupItem("blkio.throttle.io_service_bytes") {
		b := strings.Split(v, " ")
		if b[1] == "Read" {
			read += str_to_uint64(b[2])
		}
		if b[1] == "Write" {
			write += str_to_uint64(b[2])
		}
	}
	return map[string]uint64{"blkio_service_read_bytes": read, "blkio_service_write_bytes": write}, nil
}

func mem_usage(c *lxc.Container) (uint64, error) {
	if value := c.CgroupItem("memory.usage_in_bytes")[0]; value != "" {
		return str_to_uint64(value), nil
	}
	return 0, errors.New("mem_usage for the container failed")
}

func mem_limit(c *lxc.Container) (uint64, error) {
	total_memory := get_total_mem()
	if value := c.CgroupItem("memory.limit_in_bytes")[0]; value != "" {
		value_uint64 := str_to_uint64(value)
		if value_uint64 > total_memory {
			return total_memory, nil
		} else {
			return value_uint64, nil
		}
	}
	return 0, errors.New("mem_limit for the container failed")
}

func memsw_usage(c *lxc.Container) (uint64, error) {
	if value := c.CgroupItem("memory.memsw.usage_in_bytes")[0]; value != "" {
		return str_to_uint64(value), nil
	}
	return 0, errors.New("memsw_usage for the container failed")
}

func memsw_limit(c *lxc.Container) (uint64, error) {
	total_memory := get_total_mem()
	if value := c.CgroupItem("memory.memsw.limit_in_bytes")[0]; value != "" {
		value_uint64 := str_to_uint64(value)
		if value_uint64 > total_memory {
			return total_memory, nil
		} else {
			return value_uint64, nil
		}
	}
	return 0, errors.New("memsw_limit for the container failed")
}

func mem_usage_perc(mem_usage float64, mem_limit float64) (float64, error) {
	if mem_limit > 0 {
		return mem_usage / mem_limit * 100, nil
	} else {
		return 0, errors.New("mem_usage_perc for the container failed")
	}
}

func cpu_time(c *lxc.Container) (uint64, error) {
	if value := c.CgroupItem("cpuacct.usage")[0]; value != "" {
		return str_to_uint64(value), nil
	}
	return 0, errors.New("cpu_time for the container failed")
}

func cpu_time_percpu(c *lxc.Container, cpu_time float64) (float64, error) {
	if cpuset_cpus := c.CgroupItem("cpuset.cpus")[0]; cpuset_cpus != "" {
		if num_cores := count_cores(cpuset_cpus); num_cores > 0 {
			return cpu_time / float64(num_cores), nil
		}
	}
	return 0, errors.New("cpu_time_percpu for the container failed")
}

func get_total_mem() uint64 {
	virtual_mem, err := mem.VirtualMemory()
	if err != nil {
		panic("Cannot get total memory value")
	}
	return virtual_mem.Total
}

func interface_stats(c *lxc.Container) (map[string]uint64, error) {
	var iface_name string
	stats := make(map[string]map[string]uint64)

	for i := 0; i < len(c.ConfigItem("lxc.network")); i++ {
		iface_type := c.RunningConfigItem(fmt.Sprintf("lxc.network.%d.type", i))
		if iface_type == nil {
			continue
		}

		if iface_type[0] == "veth" {
			iface_name = c.RunningConfigItem(fmt.Sprintf("lxc.network.%d.veth.pair", i))[0]
		} else {
			iface_name = c.RunningConfigItem(fmt.Sprintf("lxc.network.%d.link", i))[0]
		}

		for _, v := range []string{"rx", "tx"} {
			content, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%s/statistics/%s_bytes", iface_name, v))
			if err != nil {
				return nil, err
			}

			bytes := str_to_uint64(strings.Split(string(content), "\n")[0])

			if stats[iface_name] == nil {
				stats[iface_name] = make(map[string]uint64)
			}
			stats[iface_name][v] = uint64(bytes)
		}
	}

	output := make(map[string]uint64)
	for _, value := range stats {
		output["tx"] += value["tx"]
		output["rx"] += value["rx"]
	}

	return output, nil
}

func main() {
	lxcpath := lxc.DefaultConfigPath()

	lxc_list := lxc.ActiveContainers(lxcpath)
	m := make(map[string]map[string]interface{})
	for lxc_c := range lxc_list {
		lxc_name := lxc_list[lxc_c].Name()

		c, err := lxc.NewContainer(lxc_name, lxcpath)
		if err != nil {
			panic("Cannot get LXC container statistics")
		}

		m[lxc_name] = make(map[string]interface{})

		mem_usage, err := mem_usage(c)
		if err == nil {
			m[lxc_name]["mem_usage"] = mem_usage
		}

		mem_limit, err := mem_limit(c)
		if err == nil {
			m[lxc_name]["mem_limit"] = mem_limit
		}

		_, ok_usage := m[lxc_name]["mem_usage"]
		_, ok_limit := m[lxc_name]["mem_limit"]

		if ok_usage && ok_limit {
			mem_usage_perc, err := mem_usage_perc(float64(m[lxc_name]["mem_usage"].(uint64)), float64(m[lxc_name]["mem_limit"].(uint64)))
			if err == nil {
				m[lxc_name]["mem_usage_perc"] = mem_usage_perc
			}
		}

		memsw_usage, err := memsw_usage(c)
		if err == nil {
			m[lxc_name]["memsw_usage"] = memsw_usage
		}

		memsw_limit, err := memsw_limit(c)
		if err == nil {
			m[lxc_name]["memsw_limit"] = memsw_limit
		}

		cpu_time, err := cpu_time(c)
		if err == nil {
			m[lxc_name]["cpu_time"] = cpu_time
			cpu_time_percpu, err := cpu_time_percpu(c, float64(cpu_time))
			if err == nil {
				m[lxc_name]["cpu_time_percpu"] = cpu_time_percpu
			}
		}

		blkio_serviced, err := blkio_serviced(c)
		if err == nil {
			m[lxc_name]["blkio_reads"] = blkio_serviced["blkio_serviced_read"]
			m[lxc_name]["blkio_writes"] = blkio_serviced["blkio_serviced_write"]
		}

		blkio_service_bytes, err := blkio_service_bytes(c)
		if err == nil {
			m[lxc_name]["blkio_read_bytes"] = blkio_service_bytes["blkio_service_read_bytes"]
			m[lxc_name]["blkio_write_bytes"] = blkio_service_bytes["blkio_service_write_bytes"]
		}

		ifaces_stats, err := interface_stats(c)
		if err == nil {
			/* tx and rx are reversed from the host vs container */
			m[lxc_name]["bytes_sent"] = ifaces_stats["rx"]
			m[lxc_name]["bytes_recv"] = ifaces_stats["tx"]
		}
	}

	fmt.Printf("%s\n", gen_line_prot_msg(m))
}
