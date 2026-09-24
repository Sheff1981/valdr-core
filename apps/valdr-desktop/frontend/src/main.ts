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

type WalletBalance = {
  address: string;
  balance_val: number;
  balance_vdr: string;
};

type SendPreview = {
  amount_val: number;
  amount_vdr: string;
  fee_val: number;
  fee_vdr: string;
  total_val: number;
  total_vdr: string;
};

type SendResult = SendPreview & {
  transaction_id: string;
};

type AppAPI = {
  GetState(): Promise<DesktopState>;
  CreateWallet(name: string, passphrase: string): Promise<WalletMetadata>;
  GetWalletBalance(address: string): Promise<WalletBalance>;
  PreviewSend(
    selector: string,
    passphrase: string,
    recipient: string,
    amountVDR: string,
  ): Promise<SendPreview>;
  SendTransaction(
    selector: string,
    passphrase: string,
    recipient: string,
    amountVDR: string,
  ): Promise<SendResult>;
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
        <button class="nav-item" data-view="send">Send</button>
        <button class="nav-item" data-view="receive">Receive</button>
        <button class="nav-item" data-view="wallet">Wallet</button>
        <button class="nav-item" data-view="network">Network</button>
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
        <div class="top-actions">
          <label class="wallet-select-label">
            Active wallet
            <select id="wallet-selector" aria-label="Active wallet"></select>
          </label>
          <div class="node-badge" id="node-badge">
            <span class="dot"></span>
            <span id="node-badge-text">Checking node…</span>
          </div>
        </div>
      </header>

      <section class="view active" id="view-overview">
        <div class="hero">
          <div>
            <p class="eyebrow">Spendable balance</p>
            <div class="balance"><span id="overview-balance">—</span> <span>VDR</span></div>
            <p class="subtle" id="overview-wallet-label">Create or select an encrypted wallet.</p>
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

      <section class="view" id="view-send">
        <div class="section-head">
          <div>
            <p class="eyebrow">TESTNET VDR</p>
            <h2>Send VDR</h2>
            <p class="subtle">The private key is decrypted and used only inside VALDR Desktop. The node receives only a signed transaction.</p>
          </div>
        </div>

        <div class="wallet-layout">
          <article class="card">
            <form id="send-form">
              <label>
                From
                <input id="send-from" readonly placeholder="Select a wallet">
              </label>
              <label>
                Recipient address
                <input id="send-recipient" autocomplete="off" spellcheck="false" placeholder="VDR1…" required>
              </label>
              <label>
                Amount (VDR)
                <input id="send-amount" inputmode="decimal" autocomplete="off" placeholder="0.00000000" required>
              </label>
              <label>
                Wallet passphrase
                <input id="send-passphrase" type="password" autocomplete="current-password" required>
              </label>
              <button class="primary" type="submit">Preview transaction</button>
            </form>
            <p class="warning">VALDR transactions are irreversible after broadcast. Verify the address and amount before confirming.</p>
          </article>

          <article class="card">
            <h3>Transaction preview</h3>
            <div id="send-preview-empty" class="subtle">Enter transaction details to calculate the network fee.</div>
            <div id="send-preview" class="hidden">
              <dl class="details compact">
                <div><dt>Amount</dt><dd><strong id="preview-amount">—</strong> VDR</dd></div>
                <div><dt>Network fee</dt><dd><strong id="preview-fee">—</strong> VDR</dd></div>
                <div><dt>Total spend</dt><dd><strong id="preview-total">—</strong> VDR</dd></div>
              </dl>
              <button class="primary full-button" id="confirm-send">Confirm and broadcast</button>
            </div>
            <div id="send-result" class="success-box hidden" role="status"></div>
            <div id="send-error" class="error-box hidden" role="alert"></div>
          </article>
        </div>
      </section>

      <section class="view" id="view-receive">
        <div class="section-head">
          <div>
            <p class="eyebrow">RECEIVE TESTNET VDR</p>
            <h2>Receive VDR</h2>
            <p class="subtle">Share your public VALDR address. Never share the wallet passphrase or private key.</p>
          </div>
        </div>
        <article class="card receive-card">
          <span class="subtle">Active wallet address</span>
          <code class="receive-address" id="receive-address">Select a wallet</code>
          <div class="actions">
            <button class="secondary" id="copy-address">Copy address</button>
          </div>
          <p id="copy-status" class="subtle"></p>
        </article>
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

const value = (id: string): string =>
  (document.getElementById(id) as HTMLInputElement | null)?.value ?? "";

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

const sendError = document.getElementById("send-error");
const showSendError = (message: string): void => {
  if (!sendError) return;
  sendError.textContent = message;
  sendError.classList.remove("hidden");
};
const clearSendStatus = (): void => {
  sendError?.classList.add("hidden");
  const result = document.getElementById("send-result");
  result?.classList.add("hidden");
  if (result) result.textContent = "";
};

let currentState: DesktopState | null = null;
let activeWalletAddress = "";
let activeWalletName = "";
let lastPreviewInput: {
  selector: string;
  recipient: string;
  amount: string;
} | null = null;

const activeWallet = (): WalletMetadata | undefined =>
  currentState?.wallets.find((wallet) => wallet.address === activeWalletAddress);

const renderWalletSelector = (wallets: WalletMetadata[]): void => {
  const selector = document.getElementById("wallet-selector") as HTMLSelectElement | null;
  if (!selector) return;

  const previous = activeWalletAddress;
  selector.replaceChildren();

  if (wallets.length === 0) {
    const option = document.createElement("option");
    option.value = "";
    option.textContent = "No wallet";
    selector.append(option);
    activeWalletAddress = "";
    activeWalletName = "";
    selector.disabled = true;
    return;
  }

  selector.disabled = false;
  for (const wallet of wallets) {
    const option = document.createElement("option");
    option.value = wallet.address;
    option.textContent = wallet.name || wallet.address.slice(0, 14) + "…";
    selector.append(option);
  }

  const chosen = wallets.some((wallet) => wallet.address === previous)
    ? previous
    : wallets[0].address;
  selector.value = chosen;
  activeWalletAddress = chosen;
  activeWalletName = wallets.find((wallet) => wallet.address === chosen)?.name || "VALDR Wallet";
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
    const item = document.createElement("button");
    item.type = "button";
    item.className = "wallet-item wallet-item-button";
    if (wallet.address === activeWalletAddress) {
      item.classList.add("selected");
    }

    const identity = document.createElement("div");
    const name = document.createElement("strong");
    name.textContent = wallet.name || "VALDR Wallet";
    const address = document.createElement("code");
    address.textContent = wallet.address;
    identity.append(name, address);

    const created = document.createElement("small");
    created.textContent = new Date(wallet.created_at).toLocaleDateString();

    item.append(identity, created);
    item.addEventListener("click", () => {
      activeWalletAddress = wallet.address;
      activeWalletName = wallet.name || "VALDR Wallet";
      const selector = document.getElementById("wallet-selector") as HTMLSelectElement | null;
      if (selector) selector.value = wallet.address;
      lastPreviewInput = null;
      void refreshWalletPresentation();
      renderWallets(wallets);
    });
    list.append(item);
  }
};

const refreshWalletPresentation = async (): Promise<void> => {
  const wallet = activeWallet();
  text("send-from", wallet ? wallet.address : "");
  const sendFrom = document.getElementById("send-from") as HTMLInputElement | null;
  if (sendFrom) sendFrom.value = wallet?.address ?? "";
  text("receive-address", wallet?.address ?? "Select a wallet");
  text("overview-wallet-label", wallet ? activeWalletName : "Create or select an encrypted wallet.");

  if (!wallet || !currentState?.node_status) {
    text("overview-balance", "—");
    return;
  }

  try {
    const balance = await api().GetWalletBalance(wallet.address);
    text("overview-balance", balance.balance_vdr);
  } catch {
    text("overview-balance", "—");
  }
};

const renderState = (state: DesktopState): void => {
  currentState = state;
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

  renderWalletSelector(state.wallets);
  renderWallets(state.wallets);
  void refreshWalletPresentation();
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

document.getElementById("wallet-selector")?.addEventListener("change", (event) => {
  const selector = event.currentTarget as HTMLSelectElement;
  activeWalletAddress = selector.value;
  activeWalletName = activeWallet()?.name || "VALDR Wallet";
  lastPreviewInput = null;
  clearSendStatus();
  document.getElementById("send-preview")?.classList.add("hidden");
  document.getElementById("send-preview-empty")?.classList.remove("hidden");
  if (currentState) renderWallets(currentState.wallets);
  void refreshWalletPresentation();
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

  const name = value("wallet-name");
  const pass = value("wallet-passphrase");
  const confirm = value("wallet-passphrase-confirm");

  if (!pass) {
    showError("Wallet passphrase is required.");
    return;
  }
  if (pass !== confirm) {
    showError("Passphrases do not match.");
    return;
  }

  try {
    const created = await api().CreateWallet(name, pass);
    activeWalletAddress = created.address;
    activeWalletName = created.name || "VALDR Wallet";
    const passInput = document.getElementById("wallet-passphrase") as HTMLInputElement | null;
    const confirmInput = document.getElementById("wallet-passphrase-confirm") as HTMLInputElement | null;
    if (passInput) passInput.value = "";
    if (confirmInput) confirmInput.value = "";
    await refresh();
  } catch (error) {
    showError(error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("send-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  clearSendStatus();

  const wallet = activeWallet();
  if (!wallet) {
    showSendError("Select or create a wallet first.");
    return;
  }

  const recipient = value("send-recipient").trim();
  const amount = value("send-amount").trim();
  const passphrase = value("send-passphrase");
  if (!recipient || !amount || !passphrase) {
    showSendError("Recipient, amount and wallet passphrase are required.");
    return;
  }

  try {
    const preview = await api().PreviewSend(
      wallet.address,
      passphrase,
      recipient,
      amount,
    );
    lastPreviewInput = {
      selector: wallet.address,
      recipient,
      amount,
    };
    text("preview-amount", preview.amount_vdr);
    text("preview-fee", preview.fee_vdr);
    text("preview-total", preview.total_vdr);
    document.getElementById("send-preview-empty")?.classList.add("hidden");
    document.getElementById("send-preview")?.classList.remove("hidden");
  } catch (error) {
    lastPreviewInput = null;
    showSendError(error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("confirm-send")?.addEventListener("click", async () => {
  clearSendStatus();
  const input = lastPreviewInput;
  const passphrase = value("send-passphrase");
  if (!input || !passphrase) {
    showSendError("Preview the transaction again before broadcasting.");
    return;
  }

  const recipientNow = value("send-recipient").trim();
  const amountNow = value("send-amount").trim();
  if (recipientNow !== input.recipient || amountNow !== input.amount) {
    lastPreviewInput = null;
    document.getElementById("send-preview")?.classList.add("hidden");
    document.getElementById("send-preview-empty")?.classList.remove("hidden");
    showSendError("Transaction details changed. Preview the fee again.");
    return;
  }

  try {
    const result = await api().SendTransaction(
      input.selector,
      passphrase,
      input.recipient,
      input.amount,
    );
    const passInput = document.getElementById("send-passphrase") as HTMLInputElement | null;
    if (passInput) passInput.value = "";
    lastPreviewInput = null;

    const box = document.getElementById("send-result");
    if (box) {
      box.textContent = `Broadcast accepted. Transaction ID: ${result.transaction_id}`;
      box.classList.remove("hidden");
    }
    document.getElementById("send-preview")?.classList.add("hidden");
    document.getElementById("send-preview-empty")?.classList.remove("hidden");
    await refresh();
  } catch (error) {
    showSendError(error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("copy-address")?.addEventListener("click", async () => {
  const wallet = activeWallet();
  if (!wallet) {
    text("copy-status", "Select a wallet first.");
    return;
  }
  try {
    await navigator.clipboard.writeText(wallet.address);
    text("copy-status", "Address copied.");
  } catch {
    text("copy-status", "Clipboard unavailable. Select and copy the address manually.");
  }
});

void refresh();
window.setInterval(() => {
  void refresh();
}, 5000);
