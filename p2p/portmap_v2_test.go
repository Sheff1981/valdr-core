package p2p

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestRequestPCPMap(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	oldPort := portMapServerPort
	portMapServerPort = server.LocalAddr().(*net.UDPAddr).Port
	defer func() { portMapServerPort = oldPort }()

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 256)
		n, client, err := server.ReadFromUDP(buf)
		if err != nil {
			done <- err
			return
		}
		req := buf[:n]
		resp := make([]byte, 60)
		resp[0] = 2
		resp[1] = 0x81
		resp[3] = 0
		binary.BigEndian.PutUint32(resp[4:8], 2400)
		copy(resp[24:36], req[24:36])
		resp[36] = 6
		copy(resp[40:44], req[40:44])
		resp[54] = 0xff
		resp[55] = 0xff
		resp[56] = 8
		resp[57] = 8
		resp[58] = 8
		resp[59] = 8
		_, err = server.WriteToUDP(resp, client)
		done <- err
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var nonce [12]byte
	copy(nonce[:], []byte("0123456789ab"))
	mapping, err := requestPCPMap(ctx, net.ParseIP("127.0.0.1"), 17333, 17333, 2400, nonce, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if mapping.protocol != PortMapPCP {
		t.Fatalf("protocol=%q", mapping.protocol)
	}
	if got := mapping.externalIP.String(); got != "8.8.8.8" {
		t.Fatalf("external IP=%q", got)
	}
	if mapping.externalPort != 17333 {
		t.Fatalf("external port=%d", mapping.externalPort)
	}
	if mapping.lifetime != 2400 {
		t.Fatalf("lifetime=%d", mapping.lifetime)
	}
}

func TestRequestNATPMPMap(t *testing.T) {
	server, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	oldPort := portMapServerPort
	portMapServerPort = server.LocalAddr().(*net.UDPAddr).Port
	defer func() { portMapServerPort = oldPort }()

	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 256)
		for step := 0; step < 2; step++ {
			n, client, err := server.ReadFromUDP(buf)
			if err != nil {
				done <- err
				return
			}
			req := append([]byte(nil), buf[:n]...)
			if step == 0 {
				resp := make([]byte, 12)
				resp[0] = 0
				resp[1] = 0x80
				resp[8] = 1
				resp[9] = 1
				resp[10] = 1
				resp[11] = 1
				if _, err := server.WriteToUDP(resp, client); err != nil {
					done <- err
					return
				}
				continue
			}

			resp := make([]byte, 16)
			resp[0] = 0
			resp[1] = 0x82
			copy(resp[8:12], req[4:8])
			binary.BigEndian.PutUint32(resp[12:16], 2400)
			if _, err := server.WriteToUDP(resp, client); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	mapping, err := requestNATPMPMap(ctx, net.ParseIP("127.0.0.1"), 17333, 17333, 2400)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if mapping.protocol != PortMapNATPMP {
		t.Fatalf("protocol=%q", mapping.protocol)
	}
	if got := mapping.externalIP.String(); got != "1.1.1.1" {
		t.Fatalf("external IP=%q", got)
	}
	if mapping.externalPort != 17333 {
		t.Fatalf("external port=%d", mapping.externalPort)
	}
	if mapping.lifetime != 2400 {
		t.Fatalf("lifetime=%d", mapping.lifetime)
	}
}

func TestPublicPortMapIPRejectsPrivateAndCGNAT(t *testing.T) {
	for _, raw := range []string{
		"127.0.0.1",
		"10.0.0.1",
		"192.168.1.10",
		"100.64.0.1",
		"100.127.255.254",
	} {
		if isPublicPortMapIP(net.ParseIP(raw)) {
			t.Fatalf("%s unexpectedly accepted as publicly routable", raw)
		}
	}
	for _, raw := range []string{"1.1.1.1", "8.8.8.8"} {
		if !isPublicPortMapIP(net.ParseIP(raw)) {
			t.Fatalf("%s unexpectedly rejected", raw)
		}
	}
}
