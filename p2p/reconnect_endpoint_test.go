package p2p

import (
	"context"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
)

// A forwarded port can differ from the listener advertised in hello.
// Recovery must reuse the endpoint whose handshake actually succeeded.
func TestV2ReconnectUsesSuccessfulDialEndpoint(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil {
		t.Fatal(err)
	}
	seed := mustStartNode(t, NodeConfig{
		NodeID: "mapped-seed", ListenAddress: "127.0.0.1:0",
		AdvertiseAddress: "127.0.0.1:1", NetworkProfile: &profile, EnableV2: true,
	})
	defer seed.Close()
	client := mustStartNode(t, NodeConfig{
		NodeID: "mapped-client", ListenAddress: "127.0.0.1:0",
		NetworkProfile: &profile, EnableV2: true,
	})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Connect(ctx, seed.Address()); err != nil {
		t.Fatal(err)
	}
	waitForPeerCount(t, seed, 1)
	client.mu.RLock()
	pc := client.conns["mapped-seed"]
	client.mu.RUnlock()
	client.dropPeer("mapped-seed", pc)
	waitForPeerCount(t, seed, 0)
	result := client.MaintainOutbound(ctx)
	if result.Connected != 1 {
		t.Fatalf("failed to reconnect through working endpoint: %+v", result)
	}
	waitForPeerCount(t, seed, 1)
	waitForPeerCount(t, client, 1)
}
