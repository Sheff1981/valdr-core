package explorer

import (
	"context"\n\t"encoding/json"\n\t"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sheff1981/valdr-core/config"
	"github.com/Sheff1981/valdr-core/core/blockchain"
	"github.com/Sheff1981/valdr-core/core/mempool"
	valdrcrypto "github.com/Sheff1981/valdr-core/crypto"
	"github.com/Sheff1981/valdr-core/mining"
	"github.com/Sheff1981/valdr-core/p2p"
	"github.com/Sheff1981/valdr-core/rpc"
)

func TestExplorerLiveDevnet2RPC(t *testing.T) {
	profile, err := config.ResolveNetworkProfile(config.NetworkDevnetV02)
	if err != nil { t.Fatal(err) }
	chain, err := blockchain.NewForProfile(profile)
	if err != nil { t.Fatal(err) }

	key, err := valdrcrypto.GenerateKeyPair()
	if err != nil { t.Fatal(err) }
	address, err := valdrcrypto.AddressFromPublicKey(&key.PublicKey)
	if err != nil { t.Fatal(err) }
	if _, err := mining.MineBlock(
		chain,
		address,
		profile.GenesisTimestamp+profile.TargetBlockTimeSeconds,
		nil,
	); err != nil { t.Fatal(err) }

	node, err := p2p.NewNode(p2p.NodeConfig{
		NodeID:"explorer-live-node", ListenAddress:"127.0.0.1:0",
		NetworkProfile:&profile, EnableV2:true,
		Blockchain:chain, Mempool:mempool.New(),
	})
	if err != nil { t.Fatal(err) }

	rpcServer, err := rpc.NewServer(chain, node)
	if err != nil { t.Fatal(err) }
	nodeHTTP := httptest.NewServer(rpcServer.Handler())
	defer nodeHTTP.Close()

	server, err := NewWithIndex(
		rpc.NewClient(nodeHTTP.URL),
		t.TempDir()+"/index.json",
	)
	if err != nil { t.Fatal(err) }
	explorerHTTP := httptest.NewServer(server.Handler())
	defer explorerHTTP.Close()

	client := &http.Client{Timeout:3*time.Second}
	response, err := client.Get(explorerHTTP.URL+"/api/v1/status")
	if err != nil { t.Fatal(err) }
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status endpoint HTTP=%d", response.StatusCode)
	}
	var status StatusAPI
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status.Status.ChainID != profile.ChainID ||
		status.Status.Network != profile.Name ||
		status.Status.Height != 1 ||
		status.Status.Chainwork == "" ||
		status.Status.Target == "" {
		t.Fatalf("unexpected live status: %+v", status.Status)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var rpcStatus rpc.StatusResult
	if err := rpc.NewClient(nodeHTTP.URL).Call(ctx, rpc.MethodGetStatus, nil, &rpcStatus); err != nil {
		t.Fatal(err)
	}
	if rpcStatus.ChainID != profile.ChainID {
		t.Fatalf("RPC chain id=%s want=%s", rpcStatus.ChainID, profile.ChainID)
	}

	response, err = client.Get(explorerHTTP.URL+"/")
	if err != nil { t.Fatal(err) }
	defer response.Body.Close()
	var body strings.Builder
	if _, err := io.Copy(&body, response.Body); err != nil { t.Fatal(err) }
	if !strings.Contains(body.String(), profile.ChainID) {
		t.Fatalf("overview missing Devnet2 chain id: %s", body.String())
	}
}
