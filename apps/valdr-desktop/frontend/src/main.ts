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
  best_known_height: number;
  sync_progress: number;
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

type TransactionHistoryItem = {
  status: "pending" | "confirmed";
  direction: "received" | "sent" | "self";
  transaction_id: string;
  timestamp: number;
  amount_val: number;
  amount_vdr: string;
  fee_val: number;
  fee_vdr: string;
  block_height?: number;
  block_hash?: string;
  confirmations: number;
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
  BackupWallet(selector: string): Promise<string>;
  RestoreWallet(): Promise<WalletMetadata>;
  GetTransactionHistory(address: string): Promise<TransactionHistoryItem[]>;
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
  <div id="first-run" class="first-run hidden" aria-modal="true" role="dialog">
    <div class="first-run-card">
      <div class="brand first-run-brand">
        <div class="mark" aria-hidden="true">V</div>
        <div>
          <strong>VALDR</strong>
          <span>Desktop · Testnet</span>
        </div>
      </div>
      <p class="eyebrow">FIRST RUN</p>
      <h1>Set up your VALDR wallet</h1>
      <p class="subtle">
        VALDR Desktop runs a local validating node. The blockchain is stored on this device and disk usage grows as the network grows.
        This build connects to Testnet only; Testnet VDR has no promised monetary value.
      </p>
      <div class="first-run-network">
        <span id="first-run-node-state">Starting local node…</span>
        <strong id="first-run-sync">Waiting for status</strong>
      </div>
      <form id="first-run-form">
        <label>
          Wallet name
          <input id="first-wallet-name" maxlength="64" autocomplete="off" placeholder="My VALDR wallet">
        </label>
        <label>
          Wallet passphrase
          <input id="first-wallet-passphrase" type="password" autocomplete="new-password" required>
        </label>
        <label>
          Confirm passphrase
          <input id="first-wallet-confirm" type="password" autocomplete="new-password" required>
        </label>
        <button class="primary" type="submit">Create encrypted wallet</button>
      </form>
      <div class="first-run-divider"><span>or</span></div>
      <button class="secondary full-button" id="first-run-restore">Restore encrypted backup</button>
      <p id="first-run-error" class="warning"></p>
    </div>
  </div>

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
        <button class="nav-item" data-view="transactions">Transactions</button>
        <button class="nav-item" data-view="network">Network</button>
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
            <span>Local / best height</span>
            <strong id="height">—</strong>
          </article>
          <article class="card">
            <span>Sync</span>
            <strong id="sync-progress">—</strong>
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

      <section class="view" id="view-transactions">
        <div class="section-head">
          <div>
            <p class="eyebrow">LOCAL WALLET HISTORY</p>
            <h2>Transactions</h2>
            <p class="subtle">Confirmed history is reorg-safe. Pending transactions come from the local node mempool.</p>
          </div>
          <button class="secondary" id="refresh-history">Refresh</button>
        </div>
        <div id="history-empty" class="card subtle">Select a wallet to view transactions.</div>
        <div id="history-list" class="history-list"></div>
        <div id="history-error" class="error-box hidden" role="alert"></div>
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
            <div class="section-head">
              <div>
                <h3>Wallets on this device</h3>
                <p class="subtle">Backups stay encrypted. Restoring never imports a plaintext legacy wallet.</p>
              </div>
              <div class="actions">
                <button class="secondary" id="backup-wallet">Back up selected</button>
                <button class="secondary" id="restore-wallet">Restore backup</button>
              </div>
            </div>
            <div id="wallet-list" class="wallet-list"></div>
            <p id="wallet-action-status" class="subtle"></p>
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
let currentView = "overview";
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

const renderHistory = (items: TransactionHistoryItem[]): void => {
  const list = document.getElementById("history-list");
  const empty = document.getElementById("history-empty");
  if (!list || !empty) return;
  list.replaceChildren();

  if (items.length === 0) {
    empty.textContent = activeWallet()
      ? "No transactions for this wallet yet."
      : "Select a wallet to view transactions.";
    empty.classList.remove("hidden");
    return;
  }
  empty.classList.add("hidden");

  for (const item of items) {
    const card = document.createElement("article");
    card.className = "card history-item";

    const head = document.createElement("div");
    head.className = "history-head";
    const identity = document.createElement("div");
    const direction = document.createElement("strong");
    direction.className = "history-direction " + item.direction;
    direction.textContent =
      item.direction === "received"
        ? "Received"
        : item.direction === "sent"
          ? "Sent"
          : "Self transfer";
    const when = document.createElement("small");
    when.textContent = new Date(item.timestamp * 1000).toLocaleString();
    identity.append(direction, when);

    const amount = document.createElement("strong");
    amount.className = "history-amount";
    const prefix = item.direction === "received" ? "+" : item.direction === "sent" ? "−" : "";
    amount.textContent = `${prefix}${item.amount_vdr} VDR`;
    head.append(identity, amount);

    const meta = document.createElement("dl");
    meta.className = "details compact history-details";

    const addDetail = (label: string, valueText: string): void => {
      const row = document.createElement("div");
      const dt = document.createElement("dt");
      const dd = document.createElement("dd");
      dt.textContent = label;
      dd.textContent = valueText;
      row.append(dt, dd);
      meta.append(row);
    };

    addDetail("Status", item.status);
    addDetail(
      "Confirmations",
      item.status === "pending" ? "0" : String(item.confirmations),
    );
    if (item.fee_val > 0) addDetail("Network fee", item.fee_vdr + " VDR");
    if (item.block_height) addDetail("Block", String(item.block_height));
    addDetail("Transaction ID", item.transaction_id);

    card.append(head, meta);
    list.append(card);
  }
};

const refreshTransactionHistory = async (): Promise<void> => {
  const errorBox = document.getElementById("history-error");
  errorBox?.classList.add("hidden");
  const wallet = activeWallet();
  if (!wallet || !currentState?.node_status) {
    renderHistory([]);
    return;
  }
  try {
    const items = await api().GetTransactionHistory(wallet.address);
    renderHistory(items);
  } catch (error) {
    if (errorBox) {
      errorBox.textContent = error instanceof Error ? error.message : String(error);
      errorBox.classList.remove("hidden");
    }
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

  text(
    "height",
    status ? `${status.height} / ${status.best_known_height}` : "—",
  );
  const syncLabel = !status
    ? "—"
    : status.peer_count === 0
      ? "Waiting"
      : `${Math.round(status.sync_progress * 100)}%`;
  text("sync-progress", syncLabel);
  text("peers", status ? String(status.peer_count) : "—");
  text("mempool", status ? String(status.mempool_count) : "—");
  text("detail-tip", status?.tip_hash || "—");
  text("detail-chainwork", status?.chainwork || "—");

  const healthy = state.node_running && Boolean(status);
  text("node-badge-text", healthy ? "Node online" : state.node_running ? "Node starting" : "Node stopped");
  text("node-state", healthy ? "Connected to local RPC" : state.node_running ? "Starting local node…" : "Stopped");

  const badge = document.getElementById("node-badge");
  badge?.classList.toggle("healthy", healthy);

  const firstRun = document.getElementById("first-run");
  firstRun?.classList.toggle("hidden", state.wallets.length > 0);
  text(
    "first-run-node-state",
    state.node_error
      ? "Local node needs attention"
      : healthy
        ? "Local validating node online"
        : state.node_running
          ? "Starting local validating node…"
          : "Local node stopped",
  );
  text(
    "first-run-sync",
    !status
      ? "Waiting for node status"
      : status.peer_count === 0
        ? `Height ${status.height} · waiting for peers`
        : `Sync ${Math.round(status.sync_progress * 100)}% · ${status.height}/${status.best_known_height}`,
  );

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
    currentView = view;
    document.querySelectorAll(".nav-item").forEach((item) => item.classList.remove("active"));
    button.classList.add("active");
    document.querySelectorAll(".view").forEach((item) => item.classList.remove("active"));
    document.getElementById(`view-${view}`)?.classList.add("active");
    text("view-title", button.textContent?.trim() || "VALDR");
    if (view === "transactions") {
      void refreshTransactionHistory();
    }
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
  if (currentView === "transactions") {
    void refreshTransactionHistory();
  }
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

document.getElementById("first-run-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  const name = value("first-wallet-name");
  const pass = value("first-wallet-passphrase");
  const confirm = value("first-wallet-confirm");
  if (!pass) {
    text("first-run-error", "Wallet passphrase is required.");
    return;
  }
  if (pass !== confirm) {
    text("first-run-error", "Passphrases do not match.");
    return;
  }
  try {
    text("first-run-error", "");
    const created = await api().CreateWallet(name, pass);
    activeWalletAddress = created.address;
    activeWalletName = created.name || "VALDR Wallet";
    const passInput = document.getElementById("first-wallet-passphrase") as HTMLInputElement | null;
    const confirmInput = document.getElementById("first-wallet-confirm") as HTMLInputElement | null;
    if (passInput) passInput.value = "";
    if (confirmInput) confirmInput.value = "";
    await refresh();
  } catch (error) {
    text("first-run-error", error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("first-run-restore")?.addEventListener("click", async () => {
  try {
    text("first-run-error", "");
    const restored = await api().RestoreWallet();
    if (!restored.address) return;
    activeWalletAddress = restored.address;
    activeWalletName = restored.name || "VALDR Wallet";
    await refresh();
  } catch (error) {
    text("first-run-error", error instanceof Error ? error.message : String(error));
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

document.getElementById("backup-wallet")?.addEventListener("click", async () => {
  const wallet = activeWallet();
  if (!wallet) {
    text("wallet-action-status", "Select a wallet first.");
    return;
  }
  try {
    const path = await api().BackupWallet(wallet.address);
    text(
      "wallet-action-status",
      path ? "Encrypted backup saved." : "Backup cancelled.",
    );
  } catch (error) {
    text(
      "wallet-action-status",
      error instanceof Error ? error.message : String(error),
    );
  }
});

document.getElementById("restore-wallet")?.addEventListener("click", async () => {
  try {
    const restored = await api().RestoreWallet();
    if (!restored.address) {
      text("wallet-action-status", "Restore cancelled.");
      return;
    }
    activeWalletAddress = restored.address;
    activeWalletName = restored.name || "VALDR Wallet";
    text("wallet-action-status", "Encrypted wallet restored.");
    await refresh();
  } catch (error) {
    text(
      "wallet-action-status",
      error instanceof Error ? error.message : String(error),
    );
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
    if (currentView === "transactions") {
      await refreshTransactionHistory();
    }
  } catch (error) {
    showSendError(error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("refresh-history")?.addEventListener("click", async () => {
  await refreshTransactionHistory();
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
  if (currentView === "transactions") {
    void refreshTransactionHistory();
  }
}, 5000);
