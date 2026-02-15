package tcp

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/pagpeter/trackme/pkg/server"
	"github.com/pagpeter/trackme/pkg/types"
)

// TCP packet capture variables
var (
	snapshot_len int32         = 1024
	promiscuous  bool          = false
	timeout      time.Duration = 1 * time.Millisecond
	handle       *pcap.Handle
)

// ListDevices returns all capture-capable interfaces detected by pcap.
func ListDevices() ([]pcap.Interface, error) {
	return pcap.FindAllDevs()
}

func resolveTargetIP(bindHost string) net.IP {
	host := strings.TrimSpace(bindHost)
	if host != "" && host != "0.0.0.0" && host != "::" {
		if ip := net.ParseIP(host); ip != nil {
			return ip
		}
	}

	// Resolve primary outbound IP. We don't send data; Dial picks a route/interface.
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return nil
	}
	defer conn.Close()

	if local, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return local.IP
	}
	return nil
}

func ipEquals(a, b net.IP) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Equal(b) {
		return true
	}
	a4, b4 := a.To4(), b.To4()
	return a4 != nil && b4 != nil && a4.Equal(b4)
}

// AutoDetectDevice picks a likely capture interface for the configured bind host.
func AutoDetectDevice(bindHost string) (string, error) {
	devices, err := ListDevices()
	if err != nil {
		return "", fmt.Errorf("failed to list pcap devices: %w", err)
	}
	if len(devices) == 0 {
		return "", fmt.Errorf("no pcap devices found")
	}

	targetIP := resolveTargetIP(bindHost)
	if targetIP != nil {
		for _, dev := range devices {
			for _, addr := range dev.Addresses {
				if ipEquals(addr.IP, targetIP) {
					return dev.Name, nil
				}
			}
		}
	}

	for _, dev := range devices {
		for _, addr := range dev.Addresses {
			if addr.IP != nil && !addr.IP.IsLoopback() {
				return dev.Name, nil
			}
		}
	}

	return devices[0].Name, nil
}

func parseIP(packet gopacket.Packet) *types.IPDetails {
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer == nil {
		if ipLayer := packet.Layer(layers.LayerTypeIPv6); ipLayer == nil {
			return nil
		} else {
			// IPv6
			ip := packet.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
			return &types.IPDetails{
				DstIp:     ip.DstIP.String(),
				SrcIP:     ip.SrcIP.String(),
				TTL:       int(ip.HopLimit),
				IPVersion: 6,
			}
		}
	} else {
		// IPv4
		ip := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
		return &types.IPDetails{
			DstIp:     ip.DstIP.String(),
			SrcIP:     ip.SrcIP.String(),
			ID:        int(ip.Id),
			TOS:       int(ip.TOS),
			TTL:       int(ip.TTL),
			IPVersion: 4,
		}
	}
}

func SniffTCP(device string, tlsPort int, srv *server.Server) {
	handle, err := pcap.OpenLive(device, snapshot_len, promiscuous, timeout)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			ip := parseIP(packet)
			tcp := tcpLayer.(*layers.TCP)
			if !tcp.ACK || int(tcp.DstPort) != tlsPort || ip.IPVersion == 0 {
				continue
			}

			pack := types.TCPIPDetails{
				CapLen:  packet.Metadata().CaptureLength,
				DstPort: int(tcp.DstPort),
				SrcPort: int(tcp.SrcPort),
				IP:      *ip,
				TCP: types.TCPDetails{
					Ack:          int(tcp.Ack),
					Checksum:     int(tcp.Checksum),
					Options:      parseTCPOptions(tcp.Options),
					OptionsOrder: parseTCPOptionsOrder(tcp.Options),
					Seq:          int(tcp.Seq),
					Window:       int(tcp.Window),
				},
			}
			src := net.JoinHostPort(pack.IP.SrcIP, strconv.Itoa(pack.SrcPort))
			srv.GetTCPFingerprints().Store(src, pack)
		}
	}
}

func parseTCPOptions(_ []layers.TCPOption) string {
	return ""
}

func parseTCPOptionsOrder(_ []layers.TCPOption) string {
	return ""
}
