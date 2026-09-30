package p2p

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var portMapServerPort = 5351

const (
	portMapRequestedLifetime = 40 * time.Minute
	portMapRetryPeriod       = 5 * time.Minute
	portMapRenewMax          = 20 * time.Minute
	portMapUDPTimeout        = time.Second
	portMapUDPTries          = 3
)

type PortMapProtocol string

const (
	PortMapNone   PortMapProtocol = "none"
	PortMapPCP    PortMapProtocol = "pcp"
	PortMapNATPMP PortMapProtocol = "nat-pmp"
)

type PortMapState struct {
	Protocol         PortMapProtocol `json:"protocol"`
	Active           bool            `json:"active"`
	Gateway          string          `json:"gateway,omitempty"`
	ExternalEndpoint string          `json:"external_endpoint,omitempty"`
	LifetimeSeconds  uint32          `json:"lifetime_seconds,omitempty"`
	LastError        string          `json:"last_error,omitempty"`
	UpdatedAtUTC     string          `json:"updated_at_utc"`
}

type portMapping struct {
	protocol     PortMapProtocol
	gateway      net.IP
	internalPort uint16
	externalIP   net.IP
	externalPort uint16
	lifetime     uint32
	nonce        [12]byte
}

type AutoPortMapper struct {
	port uint16

	mu    sync.RWMutex
	state PortMapState
}

func NewAutoPortMapper(port uint16) *AutoPortMapper {
	return &AutoPortMapper{
		port: port,
		state: PortMapState{
			Protocol:     PortMapNone,
			UpdatedAtUTC: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func (m *AutoPortMapper) State() PortMapState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *AutoPortMapper) setState(state PortMapState, update func(PortMapState)) {
	state.UpdatedAtUTC = time.Now().UTC().Format(time.RFC3339)
	m.mu.Lock()
	m.state = state
	m.mu.Unlock()
	if update != nil {
		update(state)
	}
}

func (m *AutoPortMapper) Run(ctx context.Context, update func(PortMapState)) {
	if m == nil || m.port == 0 {
		return
	}

	var active *portMapping
	defer func() {
		if active != nil {
			_ = removePortMapping(context.Background(), *active)
		}
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		gateway, err := queryDefaultGatewayIPv4(ctx)
		if err != nil {
			m.setState(PortMapState{
				Protocol:  PortMapNone,
				LastError: err.Error(),
			}, update)
			if !waitPortMap(ctx, portMapRetryPeriod) {
				return
			}
			continue
		}

		mapping, err := establishPortMapping(ctx, gateway, m.port)
		if err != nil {
			m.setState(PortMapState{
				Protocol:  PortMapNone,
				Gateway:   gateway.String(),
				LastError: err.Error(),
			}, update)
			if !waitPortMap(ctx, portMapRetryPeriod) {
				return
			}
			continue
		}
		active = &mapping

		for {
			state := PortMapState{
				Protocol:         mapping.protocol,
				Active:           true,
				Gateway:          gateway.String(),
				ExternalEndpoint: net.JoinHostPort(mapping.externalIP.String(), strconv.Itoa(int(mapping.externalPort))),
				LifetimeSeconds:  mapping.lifetime,
			}
			m.setState(state, update)

			renewAfter := time.Duration(mapping.lifetime) * time.Second / 2
			if renewAfter <= 0 || renewAfter > portMapRenewMax {
				renewAfter = portMapRenewMax
			}
			if !waitPortMap(ctx, renewAfter) {
				return
			}

			renewed, err := renewPortMapping(ctx, mapping)
			if err != nil {
				m.setState(PortMapState{
					Protocol:  mapping.protocol,
					Gateway:   gateway.String(),
					LastError: "renew failed: " + err.Error(),
				}, update)
				active = nil
				if !waitPortMap(ctx, portMapRetryPeriod) {
					return
				}
				break
			}
			mapping = renewed
			active = &mapping
		}
	}
}

func waitPortMap(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func establishPortMapping(ctx context.Context, gateway net.IP, port uint16) (portMapping, error) {
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return portMapping{}, err
	}
	lifetime := uint32(portMapRequestedLifetime / time.Second)

	pcpMapping, pcpErr := requestPCPMap(ctx, gateway, port, port, lifetime, nonce, nil)
	if pcpErr == nil {
		return pcpMapping, nil
	}

	natMapping, natErr := requestNATPMPMap(ctx, gateway, port, port, lifetime)
	if natErr == nil {
		return natMapping, nil
	}
	return portMapping{}, fmt.Errorf("PCP failed: %v; NAT-PMP failed: %v", pcpErr, natErr)
}

func renewPortMapping(ctx context.Context, current portMapping) (portMapping, error) {
	lifetime := uint32(portMapRequestedLifetime / time.Second)
	switch current.protocol {
	case PortMapPCP:
		return requestPCPMap(
			ctx,
			current.gateway,
			current.internalPort,
			current.externalPort,
			lifetime,
			current.nonce,
			current.externalIP,
		)
	case PortMapNATPMP:
		return requestNATPMPMap(
			ctx,
			current.gateway,
			current.internalPort,
			current.externalPort,
			lifetime,
		)
	default:
		return portMapping{}, errors.New("unknown port mapping protocol")
	}
}

func removePortMapping(ctx context.Context, current portMapping) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	switch current.protocol {
	case PortMapPCP:
		_, err := requestPCPMap(
			ctx,
			current.gateway,
			current.internalPort,
			current.externalPort,
			0,
			current.nonce,
			current.externalIP,
		)
		return err
	case PortMapNATPMP:
		_, err := requestNATPMPMap(
			ctx,
			current.gateway,
			current.internalPort,
			current.externalPort,
			0,
		)
		return err
	default:
		return nil
	}
}

func requestPCPMap(
	ctx context.Context,
	gateway net.IP,
	internalPort uint16,
	suggestedExternalPort uint16,
	lifetime uint32,
	nonce [12]byte,
	suggestedExternalIP net.IP,
) (portMapping, error) {
	conn, localIP, err := portMapUDPConn(ctx, gateway)
	if err != nil {
		return portMapping{}, err
	}
	defer conn.Close()

	request := make([]byte, 60)
	request[0] = 2 // PCP version, RFC6887.
	request[1] = 1 // MAP opcode, request bit clear.
	binary.BigEndian.PutUint32(request[4:8], lifetime)
	if err := writePCPAddress(request[8:24], localIP); err != nil {
		return portMapping{}, err
	}
	copy(request[24:36], nonce[:])
	request[36] = 6 // TCP.
	binary.BigEndian.PutUint16(request[40:42], internalPort)
	binary.BigEndian.PutUint16(request[42:44], suggestedExternalPort)
	if suggestedExternalIP != nil {
		if err := writePCPAddress(request[44:60], suggestedExternalIP); err != nil {
			return portMapping{}, err
		}
	}

	response, err := exchangePortMapUDP(ctx, conn, request, func(packet []byte) bool {
		return len(packet) >= 60 && packet[0] == 2 && packet[1] == 0x81
	})
	if err != nil {
		return portMapping{}, err
	}
	if response[3] != 0 {
		return portMapping{}, fmt.Errorf("PCP result code %d", response[3])
	}
	if !bytes.Equal(response[24:36], nonce[:]) {
		return portMapping{}, errors.New("PCP mapping nonce mismatch")
	}
	if binary.BigEndian.Uint16(response[40:42]) != internalPort {
		return portMapping{}, errors.New("PCP internal port mismatch")
	}

	externalIP, err := readPCPAddress(response[44:60])
	if err != nil {
		return portMapping{}, err
	}
	if !isPublicPortMapIP(externalIP) {
		return portMapping{}, fmt.Errorf("PCP returned non-routable external IP %s", externalIP)
	}
	externalPort := binary.BigEndian.Uint16(response[42:44])
	if externalPort == 0 {
		return portMapping{}, errors.New("PCP returned zero external port")
	}
	granted := binary.BigEndian.Uint32(response[4:8])
	if lifetime > 0 && granted == 0 {
		return portMapping{}, errors.New("PCP returned zero mapping lifetime")
	}

	return portMapping{
		protocol:     PortMapPCP,
		gateway:      append(net.IP(nil), gateway...),
		internalPort: internalPort,
		externalIP:   externalIP,
		externalPort: externalPort,
		lifetime:     granted,
		nonce:        nonce,
	}, nil
}

func requestNATPMPMap(
	ctx context.Context,
	gateway net.IP,
	internalPort uint16,
	suggestedExternalPort uint16,
	lifetime uint32,
) (portMapping, error) {
	conn, _, err := portMapUDPConn(ctx, gateway)
	if err != nil {
		return portMapping{}, err
	}
	defer conn.Close()

	externalResponse, err := exchangePortMapUDP(ctx, conn, []byte{0, 0}, func(packet []byte) bool {
		return len(packet) >= 12 && packet[0] == 0 && packet[1] == 0x80
	})
	if err != nil {
		return portMapping{}, err
	}
	if result := binary.BigEndian.Uint16(externalResponse[2:4]); result != 0 {
		return portMapping{}, fmt.Errorf("NAT-PMP external-address result code %d", result)
	}
	externalIP := net.IPv4(
		externalResponse[8],
		externalResponse[9],
		externalResponse[10],
		externalResponse[11],
	).To4()
	if !isPublicPortMapIP(externalIP) {
		return portMapping{}, fmt.Errorf("NAT-PMP returned non-routable external IP %s", externalIP)
	}

	request := make([]byte, 12)
	request[0] = 0
	request[1] = 2 // TCP mapping.
	binary.BigEndian.PutUint16(request[4:6], internalPort)
	binary.BigEndian.PutUint16(request[6:8], suggestedExternalPort)
	binary.BigEndian.PutUint32(request[8:12], lifetime)

	response, err := exchangePortMapUDP(ctx, conn, request, func(packet []byte) bool {
		return len(packet) >= 16 && packet[0] == 0 && packet[1] == 0x82
	})
	if err != nil {
		return portMapping{}, err
	}
	if result := binary.BigEndian.Uint16(response[2:4]); result != 0 {
		return portMapping{}, fmt.Errorf("NAT-PMP map result code %d", result)
	}
	if binary.BigEndian.Uint16(response[8:10]) != internalPort {
		return portMapping{}, errors.New("NAT-PMP internal port mismatch")
	}
	externalPort := binary.BigEndian.Uint16(response[10:12])
	if externalPort == 0 {
		return portMapping{}, errors.New("NAT-PMP returned zero external port")
	}
	granted := binary.BigEndian.Uint32(response[12:16])
	if lifetime > 0 && granted == 0 {
		return portMapping{}, errors.New("NAT-PMP returned zero mapping lifetime")
	}

	return portMapping{
		protocol:     PortMapNATPMP,
		gateway:      append(net.IP(nil), gateway...),
		internalPort: internalPort,
		externalIP:   append(net.IP(nil), externalIP...),
		externalPort: externalPort,
		lifetime:     granted,
	}, nil
}

func portMapUDPConn(ctx context.Context, gateway net.IP) (*net.UDPConn, net.IP, error) {
	if gateway == nil || gateway.To4() == nil {
		return nil, nil, errors.New("IPv4 default gateway unavailable")
	}
	dialer := net.Dialer{}
	connRaw, err := dialer.DialContext(
		ctx,
		"udp4",
		net.JoinHostPort(gateway.String(), strconv.Itoa(portMapServerPort)),
	)
	if err != nil {
		return nil, nil, err
	}
	conn, ok := connRaw.(*net.UDPConn)
	if !ok {
		_ = connRaw.Close()
		return nil, nil, errors.New("unexpected UDP connection type")
	}
	local, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || local.IP == nil || local.IP.To4() == nil {
		_ = conn.Close()
		return nil, nil, errors.New("could not determine local IPv4 address toward gateway")
	}
	return conn, local.IP.To4(), nil
}

func exchangePortMapUDP(
	ctx context.Context,
	conn *net.UDPConn,
	request []byte,
	accept func([]byte) bool,
) ([]byte, error) {
	buffer := make([]byte, 1100)
	var lastErr error
	for attempt := 0; attempt < portMapUDPTries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		deadline := time.Now().Add(portMapUDPTimeout)
		if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
			deadline = ctxDeadline
		}
		_ = conn.SetDeadline(deadline)
		if _, err := conn.Write(request); err != nil {
			return nil, err
		}
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				lastErr = err
				break
			}
			packet := append([]byte(nil), buffer[:n]...)
			if accept(packet) {
				return packet, nil
			}
		}
	}
	if lastErr == nil {
		lastErr = errors.New("no response")
	}
	return nil, lastErr
}

func writePCPAddress(dst []byte, ip net.IP) error {
	if len(dst) != 16 {
		return errors.New("invalid PCP address buffer")
	}
	for i := range dst {
		dst[i] = 0
	}
	if v4 := ip.To4(); v4 != nil {
		dst[10] = 0xff
		dst[11] = 0xff
		copy(dst[12:16], v4)
		return nil
	}
	if v6 := ip.To16(); v6 != nil {
		copy(dst, v6)
		return nil
	}
	return errors.New("invalid PCP IP address")
}

func readPCPAddress(src []byte) (net.IP, error) {
	if len(src) != 16 {
		return nil, errors.New("invalid PCP address")
	}
	if bytes.Equal(src[:10], make([]byte, 10)) && src[10] == 0xff && src[11] == 0xff {
		return net.IPv4(src[12], src[13], src[14], src[15]).To4(), nil
	}
	ip := net.IP(append([]byte(nil), src...))
	if ip.To16() == nil {
		return nil, errors.New("invalid PCP external address")
	}
	return ip, nil
}

func isPublicPortMapIP(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() ||
		ip.IsUnspecified() || ip.IsLoopback() || ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		// RFC6598 shared address space is not publicly reachable.
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
			return false
		}
	}
	return true
}

func queryDefaultGatewayIPv4(ctx context.Context) (net.IP, error) {
	switch runtime.GOOS {
	case "windows":
		output, err := exec.CommandContext(ctx, "route", "PRINT", "-4", "0.0.0.0").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("default gateway lookup failed: %w", err)
		}
		for _, line := range strings.Split(string(output), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 4 || fields[0] != "0.0.0.0" || fields[1] != "0.0.0.0" {
				continue
			}
			if ip := net.ParseIP(fields[2]); ip != nil && ip.To4() != nil && !ip.IsUnspecified() {
				return ip.To4(), nil
			}
		}
	case "linux":
		output, err := exec.CommandContext(ctx, "ip", "-4", "route", "show", "default").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("default gateway lookup failed: %w", err)
		}
		fields := strings.Fields(string(output))
		for i := 0; i+1 < len(fields); i++ {
			if fields[i] == "via" {
				if ip := net.ParseIP(fields[i+1]); ip != nil && ip.To4() != nil {
					return ip.To4(), nil
				}
			}
		}
	case "darwin":
		output, err := exec.CommandContext(ctx, "route", "-n", "get", "default").CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("default gateway lookup failed: %w", err)
		}
		for _, line := range strings.Split(string(output), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "gateway:") {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
			if ip := net.ParseIP(value); ip != nil && ip.To4() != nil {
				return ip.To4(), nil
			}
		}
	default:
		return nil, fmt.Errorf("automatic gateway discovery is unsupported on %s", runtime.GOOS)
	}
	return nil, errors.New("IPv4 default gateway not found")
}
