// Copyright (c) 2016-2017 Christian Saide <Supernomad>
// Licensed under the MPL-2.0, for details see https://github.com/Supernomad/quantum/blob/master/LICENSE

package common

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
)

const (
	confFile = "../dist/test/quantum.yml"
)

var (
	testPacket []byte
)

func init() {
	testPacket = make([]byte, 6)
	// IP (1.1.1.1)
	testPacket[0] = 1
	testPacket[1] = 1
	testPacket[2] = 1
	testPacket[3] = 1

	// Packet data
	testPacket[4] = 3
	testPacket[5] = 3
}

func testEq(a, b []byte) bool {

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func ExampleIPtoInt() {
	ipAddr := net.ParseIP("1.0.0.0")
	ipInt := IPtoInt(ipAddr)

	fmt.Println(ipInt)
	// Output: 1
}

func ExampleIncrementIP() {
	ipAddr := net.ParseIP("0.0.0.1")
	IncrementIP(ipAddr)

	fmt.Println(ipAddr)
	// Output: 0.0.0.2
}

func ExampleArrayEquals() {
	a := []byte{0, 1}
	b := []byte{0, 1}
	c := []byte{1, 1}

	fmt.Println(ArrayEquals(a, b), ArrayEquals(nil, nil), ArrayEquals(a, c), ArrayEquals(a, nil))
	// Output: true true false false
}

func TestArrayEquals(t *testing.T) {
	if !ArrayEquals(nil, nil) {
		t.Fatal("ArrayEquals returned false comparing nil/nil")
	}
	if ArrayEquals([]byte{0}, nil) {
		t.Fatal("ArrayEquals returned true comparing nil/non-nil")
	}
	if ArrayEquals([]byte{0, 1}, []byte{0}) {
		t.Fatal("ArrayEquals returned true comparing mismatched lengths")
	}
	if !ArrayEquals([]byte{0, 1}, []byte{0, 1}) {
		t.Fatal("ArrayEquals returned false for equal arrays")
	}
}

func TestIPtoInt(t *testing.T) {
	var expected uint32
	actual := IPtoInt(net.ParseIP("0.0.0.0"))
	if expected != actual {
		t.Fatalf("IPtoInt did not return the right value, got: %d, expected: %d", actual, expected)
	}
}

func TestIncrementIP(t *testing.T) {
	expected := net.ParseIP("10.0.0.1")

	actual := net.ParseIP("10.0.0.0")
	IncrementIP(actual)

	if !testEq(expected, actual) {
		t.Fatalf("IncrementIP did not return the right value, got: %s, expected: %s", actual, expected)
	}
}

func TestNewConfig(t *testing.T) {
	os.Setenv("QUANTUM_DEVICE_NAME", "different")
	os.Setenv("QUANTUM_LISTEN_PORT", "1")
	os.Setenv("QUANTUM_CONF_FILE", confFile)
	os.Setenv("QUANTUM_PID_FILE", "../quantum.pid")
	os.Setenv("_QUANTUM_REAL_DEVICE_NAME_", "quantum0")

	os.Args = append(os.Args, "-n", "100", "--datastore-prefix", "woot", "--datastore-tls-skip-verify", "-6", "fd00:dead:beef::2")
	cfg, err := NewConfig(NewLogger(NoopLogger))
	if err != nil {
		t.Fatalf("NewConfig returned an error, %s", err)
	}
	if cfg == nil {
		t.Fatal("NewConfig returned a blank config")
	}
	if cfg.DeviceName != "different" {
		t.Fatalf("NewConfig didn't pick up the environment variable replacement for DeviceName")
	}
	if cfg.ListenPort != 1 {
		t.Fatalf("NewConfig didn't pick up the environment variable replacement for ListenPort")
	}
	if cfg.DatastorePassword != "Password1" {
		t.Fatalf("NewConfig didn't pick up the config file replacement for Password")
	}
	if cfg.DatastorePrefix != "woot" {
		t.Fatal("NewConfig didn't pick up the cli replacement for Prefix")
	}
	if cfg.NumWorkers != runtime.NumCPU() {
		t.Fatal("NewConfig didn't pick up the cli replacement for NumWorkers")
	}
	if !cfg.DatastoreTLSSkipVerify {
		t.Fatal("NewConfig didn't pick up the cli replacement for DatastoreTLSSkipVerify")
	}

	cfg.usage(false)
	cfg.version(false)
}

func TestNewMapping(t *testing.T) {
	cfg := &Config{
		PrivateIP:  net.ParseIP("0.0.0.0"),
		PublicIPv4: net.ParseIP("1.1.1.1"),
		PublicIPv6: net.ParseIP("dead::beef"),
		ListenPort: 80,
		MachineID:  "123456",
	}

	actual := NewMapping(cfg)
	if !testEq(actual.IPv4, cfg.PublicIPv4) || !testEq(actual.IPv6, cfg.PublicIPv6) || actual.Port != cfg.ListenPort || !testEq(actual.PrivateIP, cfg.PrivateIP) {
		t.Fatalf("NewMapping did not return the right value, got: %v", actual)
	}
}

func TestParseMapping(t *testing.T) {
	cfg := &Config{
		PrivateIP:     net.ParseIP("0.0.0.0"),
		PublicIPv4:    net.ParseIP("1.1.1.1"),
		IsIPv4Enabled: true,
		PublicIPv6:    net.ParseIP("dead::beef"),
		IsIPv6Enabled: true,
		ListenPort:    80,
		MachineID:     "123456",
	}

	expected := NewMapping(cfg)
	actual, err := ParseMapping(expected.String(), cfg)
	if err != nil {
		t.Fatalf("Error occurred during test: %s", err)
	}
	if !testEq(actual.IPv4, expected.IPv4) || actual.Port != expected.Port || !testEq(actual.PrivateIP, expected.PrivateIP) {
		t.Fatalf("ParseMapping did not return the right value, got: %v, expected: %v", actual, expected)
	}
}

func TestParseNetworkConfig(t *testing.T) {
	defaultLeaseTime, _ := time.ParseDuration("48h")
	DefaultNetworkConfig := &NetworkConfig{
		Backend:     "udp",
		Network:     "10.99.0.0/16",
		StaticRange: "10.99.0.0/23",
		LeaseTime:   defaultLeaseTime,
	}

	baseIP, ipnet, _ := net.ParseCIDR(DefaultNetworkConfig.Network)
	DefaultNetworkConfig.BaseIP = baseIP
	DefaultNetworkConfig.IPNet = ipnet

	_, staticNet, _ := net.ParseCIDR(DefaultNetworkConfig.StaticRange)
	DefaultNetworkConfig.StaticNet = staticNet

	actual, err := ParseNetworkConfig(DefaultNetworkConfig.Bytes())
	if err != nil {
		t.Fatal("ParseNetworkConfig returned an error:", err)
	}
	if actual.Network != DefaultNetworkConfig.Network || actual.LeaseTime != DefaultNetworkConfig.LeaseTime {
		t.Fatalf("ParseNetworkConfig returned the wrong value, got: %v, expected: %v", actual, DefaultNetworkConfig)
	}
}

func TestParseNetworkConfigOnlyNetwork(t *testing.T) {
	netCfg := &NetworkConfig{Network: "10.99.0.0/16"}
	actual, err := ParseNetworkConfig(netCfg.Bytes())
	if err != nil {
		t.Fatal("ParseNetworkConfig returned an error:", err)
	}
	if actual.Network != netCfg.Network || actual.LeaseTime != 48*time.Hour {
		t.Fatalf("ParseNetworkConfig returned the wrong value, got: %v, expected: %v", actual, netCfg)
	}
}

func TestParseNetworkConfigIncorrectFormat(t *testing.T) {
	netCfg := &NetworkConfig{Network: "10.99.0."}
	_, err := ParseNetworkConfig(netCfg.Bytes())
	if err == nil {
		t.Fatal("ParseNetworkConfig should have errored")
	}

	netCfg.Network = "10.99.0.0/16"
	netCfg.StaticRange = "10.99.0./23"

	_, err = ParseNetworkConfig(netCfg.Bytes())
	if err == nil {
		t.Fatal("ParseNetworkConfig should have errored")
	}

	netCfg.StaticRange = "10.20.0.0/23"
	_, err = ParseNetworkConfig(netCfg.Bytes())
	if err == nil {
		t.Fatal("ParseNetworkConfig should have errored")
	}
}

func TestNewTunPayload(t *testing.T) {
	payload := NewTunPayload(testPacket, 2)
	for i := 0; i < 4; i++ {
		if payload.IPAddress[i] != 1 {
			t.Fatal("NewTunPayload returned an incorrect IP address mapping.")
		}
	}

	for i := 0; i < 2; i++ {
		if payload.Packet[i] != 3 {
			t.Fatal("NewTunPayload returned an incorrect Packet mapping.")
		}
	}
}

func TestNewSockPayload(t *testing.T) {
	payload := NewSockPayload(testPacket, 6)
	for i := 0; i < 4; i++ {
		if payload.IPAddress[i] != 1 {
			t.Fatal("NewTunPayload returned an incorrect IP address mapping.")
		}
	}

	for i := 0; i < 2; i++ {
		if payload.Packet[i] != 3 {
			t.Fatal("NewTunPayload returned an incorrect Packet mapping.")
		}
	}
}

func TestNewStats(t *testing.T) {
	stats := NewStats(1)
	if stats.Packets != 0 {
		t.Fatalf("NewStats did not return the correct default for Packets, got: %d, expected: %d", stats.Packets, 0)
	}
	if stats.Bytes != 0 {
		t.Fatalf("NewStats did not return the correct default for Bytes, got: %d, expected: %d", stats.Bytes, 0)
	}
	if stats.Links == nil {
		t.Fatalf("NewStats did not return the correct default for Links, got: %v, expected: %v", stats.Links, make(map[string]*Stats))
	}
	str := stats.String()
	if str == "" {
		t.Fatalf("String didn't return the correct value.")
	}
}

func TestNewLogger(t *testing.T) {
	log := NewLogger(NoopLogger)
	if log.Error == nil {
		t.Fatal("NewLogger returned a nil Error log.")
	}
	if log.Warn == nil {
		t.Fatal("NewLogger returned a nil Warn log.")
	}
	if log.Info == nil {
		t.Fatal("NewLogger returned a nil Info log.")
	}
	if log.Debug == nil {
		t.Fatal("NewLogger returned a nil Debug log.")
	}
}

func TestGenerateLocalMapping(t *testing.T) {
	defaultLeaseTime, _ := time.ParseDuration("48h")
	DefaultNetworkConfig := &NetworkConfig{
		Backend:     "udp",
		Network:     "10.99.0.0/16",
		StaticRange: "10.99.0.0/23",
		LeaseTime:   defaultLeaseTime,
	}

	baseIP, ipnet, _ := net.ParseCIDR(DefaultNetworkConfig.Network)
	DefaultNetworkConfig.BaseIP = baseIP
	DefaultNetworkConfig.IPNet = ipnet

	_, staticNet, _ := net.ParseCIDR(DefaultNetworkConfig.StaticRange)
	DefaultNetworkConfig.StaticNet = staticNet

	cfg := &Config{
		PrivateIP:     net.ParseIP("10.99.0.1"),
		PublicIPv4:    net.ParseIP("192.167.0.1"),
		PublicIPv6:    net.ParseIP("fd00:dead:beef::2"),
		ListenPort:    1099,
		NetworkConfig: DefaultNetworkConfig,
		MachineID:     "123",
	}

	mappings := make(map[uint32]*Mapping)
	mapping, err := GenerateLocalMapping(cfg, mappings)
	if err != nil {
		t.Fatal(err)
	}

	if !testEq(mapping.PrivateIP.To4(), cfg.PrivateIP.To4()) {
		t.Fatal("GenerateLocalMapping created the wrong mapping.")
	}

	mappings[IPtoInt(cfg.PrivateIP)] = mapping

	_, err = GenerateLocalMapping(cfg, mappings)
	if err != nil {
		t.Fatal(err)
	}

	mapping.MachineID = "456"

	_, err = GenerateLocalMapping(cfg, mappings)
	if err == nil {
		t.Fatal("GenerateLocalMapping failed to properly handle an existing ip address")
	}

	cfg.PrivateIP = nil
	mapping.MachineID = "123"

	_, err = GenerateLocalMapping(cfg, mappings)
	if err != nil {
		t.Fatal(err)
	}

	cfg.PrivateIP = nil
	_, err = GenerateLocalMapping(cfg, make(map[uint32]*Mapping))
	if err != nil {
		t.Fatal(err)
	}
}

func TestStatsLogBytes(t *testing.T) {
	statsl := &StatsLog{
		TxStats: &Stats{},
		RxStats: &Stats{},
	}

	slice := statsl.Bytes(false)
	if slice == nil {
		t.Fatal("StatsLog Bytes returned nil slice")
	}
	str := statsl.String(true)
	if str == "" {
		t.Fatal("StatsLog String returned empty string")
	}
}

func TestSignaler(t *testing.T) {
	log := NewLogger(NoopLogger)
	cfg, err := NewConfig(log)
	signaler := NewSignaler(log, cfg, []int{1}, map[string]string{"QUANTUM_TESTING": "woot"})

	go func() {
		time.Sleep(1 * time.Second)
		signaler.signals <- syscall.SIGHUP
		time.Sleep(1 * time.Second)
		signaler.signals <- syscall.SIGINT
	}()

	err = signaler.Wait(false)
	if err != nil {
		t.Fatal("Wait returned an error: " + err.Error())
	}
	err = signaler.Wait(false)
	if err != nil {
		t.Fatal("Wait returned an error: " + err.Error())
	}
}
