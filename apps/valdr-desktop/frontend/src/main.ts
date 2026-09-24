import "./app.css";

type WalletMetadata = {
  name: string;
  address: string;
  public_key: string;
  created_at: string;
};

type NodeStatus = {
  network: string;
  chain_id: string;
  height: number;
  tip_hash: string;
  chainwork: string;
  peer_count: number;
  mempool_count: number;
};

type DesktopState = {
  network: string;
  chain_id: string;
  mainnet_enabled: boolean;
  paths: {
    root: string;
    node_data: string;
    wallets: string;
    logs: string;
    network: string;
  };
  node_running: boolean;
  node_status?: NodeStatus;
  node_error?: string;
  wallets: WalletMetadata[];
};

type AppAPI = {
  GetState(): Promise<DesktopState>;
  CreateWallet(name: string, passphrase: string): Promise<WalletMetadata>;
  StartNode(): Promise<void>;
  StopNode(): Promise<void>;
};

declare global {
  interface Window {
    go?: {
      main?: {
        App?: AppAPI;
      };
    };
  }
}

const root = document.querySelector<HTMLDivElement>("#app");
if (!root) {
  throw new Error("VALDR Desktop root element is missing");
}

root.innerHTML = `
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="mark" aria-hidden="true">V</div>
        <div>
          <strong>VALDR</strong>
          <span>Desktop</span>
        </div>
      </div>

      <nav aria-label="Primary navigation">
        <button class="nav-item active" data-view="overview">Overview</button>
        <button class="nav-item" data-view="wallet">Wallet</button>
        <button class="nav-item" data-view="network">Network</button>
        <button class="nav-item muted" disabled>Send <small>next</small></button>
        <button class="nav-item muted" disabled>Receive <small>next</small></button>
        <button class="nav-item muted" disabled>Transactions <small>next</small></button>
      </nav>

      <div class="sidebar-foot">
        <span class="testnet-pill">TESTNET</span>
        <small>Mainnet disabled</small>
      </div>
    </aside>

    <main class="content">
      <header class="topbar">
        <div>
          <p class="eyebrow">VALDR NETWORK</p>
          <h1 id="view-title">Overview</h1>
        </div>
        <div class="node-badge" id="node-badge">
          <span class="dot"></span>
          <span id="node-badge-text">Checking node…</span>
        </div>
      </header>

      <section class="view active" id="view-overview">
        <div class="hero">
          <div>
            <p class="eyebrow">Wallet balance</p>
            <div class="balance">— <span>VDR</span></div>
            <p class="subtle">Balance display activates with the Send/Receive slice.</p>
          </div>
          <div class="network-id">
            <span>Network</span>
            <strong id="network-name">Testnet</strong>
            <code id="chain-id">valdr-testnet-1</code>
          </div>
        </div>

        <div class="grid stats">
          <article class="card">
            <span>Block height</span>
            <strong id="height">—</strong>
          </article>
          <article class="card">
            <span>Peers</span>
            <strong id="peers">—</strong>
          </article>
          <article class="card">
            <span>Mempool</span>
            <strong id="mempool">—</strong>
          </article>
          <article class="card">
            <span>Wallets</span>
            <strong id="wallet-count">0</strong>
          </article>
        </div>

        <article class="card node-panel">
          <div>
            <span>Local VALDR node</span>
            <strong id="node-state">Starting…</strong>
            <p id="node-detail" class="subtle">Desktop uses outbound-only P2P by default.</p>
          </div>
          <div class="actions">
            <button class="secondary" id="start-node">Start node</button>
            <button class="danger" id="stop-node">Stop node</button>
          </div>
        </article>

        <div id="global-error" class="error-box hidden" role="alert"></div>
      </section>

      <section class="view" id="view-wallet">
        <div class="section-head">
          <div>
            <p class="eyebrow">SELF-CUSTODY</p>
            <h2>Encrypted wallets</h2>
          </div>
        </div>

        <div class="wallet-layout">
          <article class="card">
            <h3>Create wallet</h3>
            <p class="subtle">The private key is encrypted locally with wallet v2 and never sent to the node.</p>
            <form id="create-wallet-form">
              <label>
                Wallet name
                <input id="wallet-name" autocomplete="off" maxlength="64" placeholder="My VALDR wallet">
              </label>
              <label>
                Passphrase
                <input id="wallet-passphrase" type="password" autocomplete="new-password" required>
              </label>
              <label>
                Confirm passphrase
                <input id="wallet-passphrase-confirm" type="password" autocomplete="new-password" required>
              </label>
              <button class="primary" type="submit">Create encrypted wallet</button>
            </form>
            <p class="warning">VALDR cannot recover a forgotten wallet passphrase.</p>
          </article>

          <article class="card">
            <h3>Wallets on this device</h3>
            <div id="wallet-list" class="wallet-list"></div>
          </article>
        </div>
      </section>

      <section class="view" id="view-network">
        <article class="card">
          <h2>Network diagnostics</h2>
          <dl class="details">
            <div><dt>Profile</dt><dd id="detail-network">testnet</dd></div>
            <div><dt>Chain ID</dt><dd id="detail-chain">valdr-testnet-1</dd></div>
            <div><dt>Tip</dt><dd><code id="detail-tip">—</code></dd></div>
            <div><dt>Chainwork</dt><dd><code id="detail-chainwork">—</code></dd></div>
            <div><dt>Node data</dt><dd><code id="detail-data">—</code></dd></div>
            <div><dt>Wallet data</dt><dd><code id="detail-wallets">—</code></dd></div>
          </dl>
          <p class="subtle">Inbound public-node mode is intentionally not exposed in this first Desktop slice.</p>
        </article>
      </section>
    </main>
  </div>
`;

const api = (): AppAPI => {
  const instance = window.go?.main?.App;
  if (!instance) {
    throw new Error("VALDR Desktop backend is unavailable");
  }
  return instance;
};

const text = (id: string, value: string): void => {
  const element = document.getElementById(id);
  if (element) {
    element.textContent = value;
  }
};

const globalError = document.getElementById("global-error");
const showError = (message: string): void => {
  if (!globalError) return;
  globalError.textContent = message;
  globalError.classList.remove("hidden");
};
const clearError = (): void => {
  if (!globalError) return;
  globalError.textContent = "";
  globalError.classList.add("hidden");
};

const renderWallets = (wallets: WalletMetadata[]): void => {
  const list = document.getElementById("wallet-list");
  if (!list) return;
  list.replaceChildren();

  if (wallets.length === 0) {
    const empty = document.createElement("p");
    empty.className = "subtle";
    empty.textContent = "No wallet created yet.";
    list.append(empty);
    return;
  }

  for (const wallet of wallets) {
    const item = document.createElement("div");
    item.className = "wallet-item";

    const identity = document.createElement("div");
    const name = document.createElement("strong");
    name.textContent = wallet.name || "VALDR Wallet";
    const address = document.createElement("code");
    address.textContent = wallet.address;
    identity.append(name, address);

    const created = document.createElement("small");
    created.textContent = new Date(wallet.created_at).toLocaleDateString();

    item.append(identity, created);
    list.append(item);
  }
};

const renderState = (state: DesktopState): void => {
  const status = state.node_status;
  text("network-name", state.network.toUpperCase());
  text("chain-id", state.chain_id);
  text("wallet-count", String(state.wallets.length));
  text("detail-network", state.network);
  text("detail-chain", state.chain_id);
  text("detail-data", state.paths.node_data);
  text("detail-wallets", state.paths.wallets);

  text("height", status ? String(status.height) : "—");
  text("peers", status ? String(status.peer_count) : "—");
  text("mempool", status ? String(status.mempool_count) : "—");
  text("detail-tip", status?.tip_hash || "—");
  text("detail-chainwork", status?.chainwork || "—");

  const healthy = state.node_running && Boolean(status);
  text("node-badge-text", healthy ? "Node online" : state.node_running ? "Node starting" : "Node stopped");
  text("node-state", healthy ? "Connected to local RPC" : state.node_running ? "Starting local node…" : "Stopped");

  const badge = document.getElementById("node-badge");
  badge?.classList.toggle("healthy", healthy);

  if (state.node_error) {
    text("node-detail", state.node_error);
  } else if (healthy && status) {
    text("node-detail", `Outbound-only · ${status.peer_count} peer(s) · height ${status.height}`);
  } else {
    text("node-detail", "Desktop uses outbound-only P2P by default.");
  }

  renderWallets(state.wallets);
};

const refresh = async (): Promise<void> => {
  try {
    const state = await api().GetState();
    renderState(state);
  } catch (error) {
    showError(error instanceof Error ? error.message : String(error));
  }
};

document.querySelectorAll<HTMLButtonElement>(".nav-item[data-view]").forEach((button) => {
  button.addEventListener("click", () => {
    const view = button.dataset.view;
    if (!view) return;
    document.querySelectorAll(".nav-item").forEach((item) => item.classList.remove("active"));
    button.classList.add("active");
    document.querySelectorAll(".view").forEach((item) => item.classList.remove("active"));
    document.getElementById(`view-${view}`)?.classList.add("active");
    text("view-title", button.textContent?.trim() || "VALDR");
  });
});

document.getElementById("start-node")?.addEventListener("click", async () => {
  clearError();
  try {
    await api().StartNode();
    await refresh();
  } catch (error) {
    showError(error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("stop-node")?.addEventListener("click", async () => {
  clearError();
  try {
    await api().StopNode();
    await refresh();
  } catch (error) {
    showError(error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("create-wallet-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  clearError();

  const name = (document.getElementById("wallet-name") as HTMLInputElement | null)?.value ?? "";
  const pass = (document.getElementById("wallet-passphrase") as HTMLInputElement | null)?.value ?? "";
  const confirm = (document.getElementById("wallet-passphrase-confirm") as HTMLInputElement | null)?.value ?? "";

  if (!pass) {
    showError("Wallet passphrase is required.");
    return;
  }
  if (pass !== confirm) {
    showError("Passphrases do not match.");
    return;
  }

  try {
    await api().CreateWallet(name, pass);
    const passInput = document.getElementById("wallet-passphrase") as HTMLInputElement | null;
    const confirmInput = document.getElementById("wallet-passphrase-confirm") as HTMLInputElement | null;
    if (passInput) passInput.value = "";
    if (confirmInput) confirmInput.value = "";
    await refresh();
  } catch (error) {
    showError(error instanceof Error ? error.message : String(error));
  }
});

void refresh();
window.setInterval(() => {
  void refresh();
}, 5000);
