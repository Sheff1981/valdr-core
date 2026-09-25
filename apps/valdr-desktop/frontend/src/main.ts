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

type DesktopPreferences = {
  version: number;
  language: "en" | "ru";
  theme: "dark" | "classic";
  start_node: boolean;
  advanced: boolean;
  public_node: boolean;
  public_node_advertise_address: string;
  node_data_directory?: string;
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
  unlocked_wallets: string[];
  wallet_auto_lock_minutes: number;
  initialization_ready: boolean;
  preferences: DesktopPreferences;
};

type ExplorerStatus = {
  available: boolean;
  url: string;
  chain_id?: string;
  height?: number;
  tip_hash?: string;
  message?: string;
};

type StorageDiagnostics = {
  path: string;
  ready: boolean;
  file_count: number;
  size_bytes: number;
  truncated: boolean;
  error?: string;
};

type PeerInfo = {
  node_id: string;
  address: string;
  height: number;
  cumulative_chainwork?: string;
  protocol_version: number;
  inbound: boolean;
};

type MinerStatus = {
  running: boolean;
  reward_address?: string;
  accepted_blocks: number;
  last_error?: string;
  height: number;
  next_height: number;
  current_target?: string;
  block_reward_val: number;
  block_reward_vdr: string;
  target_block_time_seconds: number;
  retarget_interval: number;
  blocks_until_retarget: number;
  hashrate_hps: number;
  last_block_hashrate_hps: number;
  last_block_hashes: number;
  last_block_duration_ms: number;
  total_hashes: number;
  total_mining_duration_ms: number;
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
  UnlockWallet(selector: string, passphrase: string): Promise<WalletMetadata>;
  LockWallet(selector: string): Promise<void>;
  SetWalletAutoLockMinutes(minutes: number): Promise<void>;
  ChooseFirstRunNodeDataDirectory(): Promise<string>;
  GetReceiveQRCode(address: string): Promise<string>;
  CopyReceiveAddress(address: string): Promise<void>;
  ExportPrivateKey(selector: string, confirmation: string): Promise<string>;
  GetWalletBalance(address: string): Promise<WalletBalance>;
  GetPeers(): Promise<PeerInfo[]>;
  GetNodeLogs(): Promise<string>;
  GetStorageDiagnostics(): Promise<StorageDiagnostics>;
  GetExplorerStatus(): Promise<ExplorerStatus>;
  OpenExplorer(): Promise<void>;
  ExportDiagnostics(): Promise<string>;
  SetPublicNodeMode(enabled: boolean, advertiseAddress: string): Promise<DesktopPreferences>;
  PreviewSend(
    selector: string,
    recipient: string,
    amountVDR: string,
  ): Promise<SendPreview>;
  SendTransaction(
    selector: string,
    recipient: string,
    amountVDR: string,
  ): Promise<SendResult>;
  BackupWallet(selector: string): Promise<string>;
  RestoreWallet(passphrase: string): Promise<WalletMetadata>;
  GetTransactionHistory(address: string): Promise<TransactionHistoryItem[]>;
  SetDesktopPreferences(
    language: "en" | "ru",
    theme: "dark" | "classic",
    startNode: boolean,
    advanced: boolean,
  ): Promise<DesktopPreferences>;
  GetMiningState(): Promise<MinerStatus>;
  StartMining(rewardAddress: string): Promise<void>;
  StopMining(): Promise<void>;
  StartNode(): Promise<void>;
  RestartNode(): Promise<void>;
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
  <div id="boot-splash" class="boot-splash" aria-hidden="true">
    <div class="boot-splash-glow"></div>
    <div class="boot-splash-panel">
      <div class="boot-coin">
        <img src="/valdr-emblem.svg" alt="">
      </div>
      <div class="boot-copy">
        <p class="eyebrow">NOT A TOKEN. A CHAIN.</p>
        <h1>VALDR DESKTOP</h1>
        <p>Preparing secure wallet and Testnet environment</p>
        <div class="boot-progress"><span></span></div>
        <small>TESTNET · MAINNET DISABLED</small>
      </div>
    </div>
  </div>

  <div id="transaction-detail-dialog" class="transaction-detail-dialog hidden" aria-modal="true" role="dialog" aria-labelledby="transaction-detail-title">
    <div class="transaction-detail-card">
      <div class="section-head">
        <div>
          <p class="eyebrow">TRANSACTION DETAIL</p>
          <h2 id="transaction-detail-title">Transaction</h2>
        </div>
        <button class="secondary" id="close-transaction-detail" type="button">Close</button>
      </div>
      <dl class="details transaction-detail-fields">
        <div><dt>Status</dt><dd id="transaction-detail-status">—</dd></div>
        <div><dt>Direction</dt><dd id="transaction-detail-direction">—</dd></div>
        <div><dt>Timestamp</dt><dd id="transaction-detail-time">—</dd></div>
        <div><dt>Amount</dt><dd id="transaction-detail-amount">—</dd></div>
        <div><dt>Network fee</dt><dd id="transaction-detail-fee">—</dd></div>
        <div><dt>Confirmations</dt><dd id="transaction-detail-confirmations">—</dd></div>
        <div><dt>Block height</dt><dd id="transaction-detail-height">—</dd></div>
        <div><dt>Block hash</dt><dd><code id="transaction-detail-block">—</code></dd></div>
        <div><dt>Transaction ID</dt><dd><code id="transaction-detail-txid">—</code></dd></div>
      </dl>
    </div>
  </div>

  <div id="send-confirm-dialog" class="send-confirm-dialog hidden" aria-modal="true" role="dialog" aria-labelledby="send-confirm-title">
    <div class="send-confirm-card">
      <p class="eyebrow">FINAL CONFIRMATION</p>
      <h2 id="send-confirm-title">Broadcast transaction?</h2>
      <p class="warning">VALDR transactions are irreversible after broadcast. Verify every value before continuing.</p>
      <dl class="details compact send-confirm-details">
        <div><dt>Recipient</dt><dd><code id="confirm-recipient">—</code></dd></div>
        <div><dt>Amount</dt><dd><strong id="confirm-amount">—</strong> VDR</dd></div>
        <div><dt>Network fee</dt><dd><strong id="confirm-fee">—</strong> VDR</dd></div>
        <div><dt>Total spend</dt><dd><strong id="confirm-total">—</strong> VDR</dd></div>
      </dl>
      <div class="actions send-confirm-actions">
        <button class="danger" id="broadcast-send" type="button">Broadcast transaction</button>
        <button class="secondary" id="cancel-send-confirmation" type="button">Cancel</button>
      </div>
    </div>
  </div>

  <div id="first-run" class="first-run hidden" aria-modal="true" role="dialog">
    <div class="first-run-card">
      <div class="brand first-run-brand">
        <img class="brand-emblem" src="/valdr-emblem.svg" alt="VALDR">
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
      <p class="subtle">Requires local disk space and outbound network access. Inbound ports are not required in the default Desktop mode.</p>
      <div class="first-run-network">
        <span>Node data<br><code id="first-run-data-path">—</code></span>
        <button class="secondary" id="first-run-choose-data" type="button">Choose folder</button>
      </div>
      <div class="first-run-network">
        <span id="first-run-node-state">Local node starts after wallet setup</span>
        <strong id="first-run-sync">Waiting for encrypted wallet</strong>
      </div>
      <button class="secondary full-button hidden restart-node-action" id="first-run-restart-node" type="button">Start / restart local node</button>
      <div id="first-run-wallet-setup">
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
        <label>
          Backup passphrase
          <input id="first-run-restore-passphrase" type="password" autocomplete="current-password">
        </label>
        <button class="secondary full-button" id="first-run-restore">Restore encrypted backup</button>
      </div>
      <p id="first-run-initializing" class="subtle hidden">Encrypted wallet is ready. Waiting for the local Testnet node to finish initialization…</p>
      <p id="first-run-error" class="warning"></p>
    </div>
  </div>

  <div class="shell">
    <aside class="sidebar">
      <div class="brand brand-sidebar">
        <img class="brand-emblem brand-emblem-sidebar" src="/valdr-emblem.svg" alt="VALDR">
        <div>
          <strong>VALDR</strong>
          <span>Desktop · Testnet</span>
        </div>
      </div>

      <nav aria-label="Primary navigation">
        <button class="nav-item active" data-view="overview">Dashboard</button>
        <button class="nav-item" data-view="wallet">Wallet</button>
        <button class="nav-item" data-view="send">Send</button>
        <button class="nav-item" data-view="receive">Receive</button>
        <button class="nav-item advanced-only hidden" data-view="mining">Mining</button>
        <button class="nav-item advanced-only hidden" data-view="network">Network</button>
        <button class="nav-item" data-view="transactions">Transactions</button>
        <button class="nav-item" data-view="settings">Settings</button>
      </nav>

      <div class="sidebar-foot">
        <span class="testnet-pill">TESTNET</span>
        <small id="mainnet-status">Mainnet disabled</small>
      </div>
    </aside>

    <main class="content">
      <header class="topbar valdr-topbar">
        <div class="product-heading">
          <p class="eyebrow">NOT A TOKEN. A CHAIN.</p>
          <h1 class="product-title">VALDR DESKTOP</h1>
          <p class="product-subtitle">SECURE WALLET · STRONGER NETWORK · A BRIGHTER TOMORROW</p>
          <p class="view-label" id="view-title">Dashboard</p>
        </div>
        <div class="top-actions">
          <label class="wallet-select-label">
            Active wallet
            <select id="wallet-selector" aria-label="Active wallet"></select>
          </label>
          <div class="status-stack">
            <div class="node-badge" id="node-badge">
              <span class="dot"></span>
              <span id="node-badge-text">Checking node…</span>
            </div>
            <div class="top-network-meta">
              <span>Block <strong id="top-block">—</strong></span>
              <span>Peers <strong id="top-peers">—</strong></span>
            </div>
          </div>
        </div>
      </header>

      <section class="view active" id="view-overview">
        <div class="hero valdr-hero reference-hero">
          <div class="hero-main">
            <p class="eyebrow">TOTAL SPENDABLE BALANCE</p>
            <div class="balance"><span id="overview-balance">—</span> <span>VDR</span></div>
            <p class="subtle" id="overview-wallet-label">Create or select an encrypted wallet.</p>
            <div class="hero-actions">
              <button class="primary view-shortcut" data-go-view="send" type="button">Send VDR</button>
              <button class="secondary view-shortcut" data-go-view="receive" type="button">Receive</button>
              <button class="secondary advanced-only hidden view-shortcut" data-go-view="mining" type="button">Mining</button>
            </div>
          </div>
          <div class="network-id hero-network-id">
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
            <button class="secondary hidden restart-node-action" type="button">Restart node</button>
            <button class="danger" id="stop-node">Stop node</button>
          </div>
        </article>

        <article class="card latest-transaction">
          <div class="section-head">
            <div>
              <span>Latest transaction</span>
              <strong id="overview-latest-direction">No transactions yet</strong>
            </div>
            <strong id="overview-latest-amount">—</strong>
          </div>
          <div id="overview-latest-meta" class="subtle">Select a wallet to view its latest activity.</div>
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
              <div class="security-row compact-security">
                <span>Signing state</span>
                <strong id="send-wallet-lock-state">Locked</strong>
              </div>
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
          <div class="receive-grid">
            <div class="receive-details">
              <span class="subtle">Active wallet address</span>
              <code class="receive-address" id="receive-address">Select a wallet</code>
              <div class="actions">
                <button class="secondary" id="copy-address">Copy address</button>
              </div>
              <p id="copy-status" class="subtle"></p>
            </div>
            <div id="receive-qr-panel" class="receive-qr-panel hidden">
              <img id="receive-qr" class="receive-qr" alt="VALDR receive address QR code">
              <span>Generated locally · no web service</span>
            </div>
          </div>
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
              </div>
            </div>
            <div id="wallet-list" class="wallet-list"></div>
            <form id="restore-wallet-form" class="restore-wallet-form">
              <label>
                Restore backup passphrase
                <input id="restore-wallet-passphrase" type="password" autocomplete="current-password" required>
              </label>
              <button class="secondary" id="restore-wallet" type="submit">Restore encrypted backup</button>
            </form>
            <p id="wallet-action-status" class="subtle"></p>

            <div class="wallet-security">
              <div class="section-head">
                <div>
                  <h3>Wallet security</h3>
                  <p class="subtle">Unlocking is local to this Desktop process. The cached passphrase is erased on lock, timeout or application shutdown.</p>
                </div>
                <strong id="wallet-lock-state">Locked</strong>
              </div>
              <form id="unlock-wallet-form">
                <label>
                  Passphrase
                  <input id="unlock-wallet-passphrase" type="password" autocomplete="current-password" required>
                </label>
                <button class="primary" type="submit">Unlock selected wallet</button>
              </form>
              <div class="security-controls">
                <button class="danger" id="lock-wallet" type="button">Lock now</button>
                <label class="auto-lock-label">
                  Auto-lock after inactivity
                  <select id="wallet-auto-lock">
                    <option value="1">1 minute</option>
                    <option value="5">5 minutes</option>
                    <option value="15" selected>15 minutes</option>
                    <option value="30">30 minutes</option>
                    <option value="60">60 minutes</option>
                  </select>
                </label>
              </div>
              <p id="wallet-security-status" class="subtle"></p>

              <div class="private-key-export">
                <button class="danger" id="show-export-warning" type="button">Export private key</button>
                <div id="export-warning" class="export-warning hidden">
                  <p class="warning"><strong>High risk:</strong> anyone with this private key can spend this wallet's funds. Never send it to support, a website, or another person.</p>
                  <label>
                    Type <code>EXPORT PRIVATE KEY</code> to continue
                    <input id="export-confirmation" autocomplete="off" spellcheck="false">
                  </label>
                  <div class="actions">
                    <button class="danger" id="confirm-private-key-export" type="button">Reveal private key</button>
                    <button class="secondary" id="cancel-private-key-export" type="button">Cancel</button>
                  </div>
                </div>
                <div id="private-key-result" class="private-key-result hidden">
                  <p class="warning">Private key visible. Store it offline and hide it immediately after use.</p>
                  <code id="private-key-value"></code>
                  <button class="secondary" id="hide-private-key" type="button">Hide private key</button>
                </div>
              </div>
            </div>
          </article>
        </div>
      </section>

      <section class="view" id="view-network">
        <article class="card">
          <h2>Network diagnostics</h2>
          <dl class="details">
            <div><dt>Profile</dt><dd id="detail-network">testnet</dd></div>
            <div><dt>Chain ID</dt><dd id="detail-chain">valdr-testnet-1</dd></div>
            <div><dt>Node state</dt><dd id="detail-node-state">—</dd></div>
            <div><dt>Synchronization</dt><dd id="detail-sync">—</dd></div>
            <div><dt>Local / best height</dt><dd id="detail-height">—</dd></div>
            <div><dt>Connected peers</dt><dd id="detail-peer-count">—</dd></div>
            <div><dt>Tip</dt><dd><code id="detail-tip">—</code></dd></div>
            <div><dt>Chainwork</dt><dd><code id="detail-chainwork">—</code></dd></div>
            <div><dt>Node data</dt><dd><code id="detail-data">—</code></dd></div>
            <div><dt>Node data state</dt><dd id="detail-storage-state">—</dd></div>
            <div><dt>Node data size</dt><dd id="detail-storage-size">—</dd></div>
            <div><dt>Node data files</dt><dd id="detail-storage-files">—</dd></div>
            <div><dt>Wallet data</dt><dd><code id="detail-wallets">—</code></dd></div>
            <div><dt>Logs</dt><dd><code id="detail-logs">—</code></dd></div>
          </dl>
          <div class="explorer-panel">
            <div class="section-head">
              <div>
                <h3>Explorer</h3>
                <p class="subtle">Open the separate read-only VALDR Explorer when it is running locally on Testnet.</p>
              </div>
              <div class="actions">
                <button class="secondary" id="refresh-explorer-status" type="button">Check Explorer</button>
                <button class="secondary" id="open-explorer" type="button" disabled>Open Explorer</button>
              </div>
            </div>
            <p id="explorer-status" class="subtle">Not checked yet.</p>
          </div>
          <div class="storage-diagnostics-panel">
            <div class="section-head">
              <div>
                <h3>Storage diagnostics</h3>
                <p class="subtle">Read-only inspection of the local node data directory.</p>
              </div>
              <button class="secondary" id="refresh-storage-diagnostics" type="button">Refresh storage</button>
            </div>
            <p id="storage-diagnostics-status" class="subtle">Not inspected yet.</p>
          </div>
          <div class="public-node-panel">
            <div class="section-head">
              <div>
                <h3>Public full node</h3>
                <p class="subtle">Advanced mode only. Normal Desktop operation remains outbound-only.</p>
              </div>
            </div>
            <label class="settings-check public-node-toggle">
              <input id="public-node-enabled" type="checkbox">
              <span>Accept inbound Testnet P2P connections</span>
            </label>
            <label>
              Advertised address
              <input id="public-node-address" autocomplete="off" spellcheck="false" placeholder="node.example.org:17333">
            </label>
            <div class="actions">
              <button class="secondary" id="apply-public-node" type="button">Apply &amp; restart node</button>
            </div>
            <p class="warning">This opens the P2P listener on all local interfaces. VALDR does not change your router or firewall. Use an explicitly routable DNS/IP endpoint. RPC remains bound to localhost.</p>
            <p id="public-node-status" class="subtle"></p>
          </div>
          <div class="network-peers">
            <div class="section-head">
              <div>
                <h3>Connected peers</h3>
                <p class="subtle">Read-only view from the managed local node.</p>
              </div>
              <button class="secondary" id="refresh-peers" type="button">Refresh peers</button>
            </div>
            <div id="peer-list" class="peer-list"></div>
            <p id="peer-list-empty" class="subtle">No connected peers.</p>
          </div>
          <div class="node-log-panel">
            <div class="section-head">
              <div>
                <h3>Node logs</h3>
                <p class="subtle">Bounded local diagnostics. Sensitive-looking lines are redacted before display.</p>
              </div>
              <button class="secondary" id="refresh-node-logs" type="button">Refresh logs</button>
            </div>
            <pre id="node-log-output" class="node-log-output">No node log output yet.</pre>
          </div>
          <p class="subtle">Default mode is outbound-only. Public Testnet P2P is opt-in above and never changes the localhost RPC boundary.</p>
        </article>
      </section>

      <section class="view" id="view-mining">
        <div class="section-head">
          <div>
            <p class="eyebrow">ADVANCED · TESTNET ONLY</p>
            <h2>Mining</h2>
            <p class="subtle">VALDR Desktop starts a separate local valdr-miner process. Mining never starts automatically.</p>
          </div>
        </div>
        <div class="mining-grid">
          <article class="card">
            <h3>Miner control</h3>
            <label>
              Reward address
              <input id="mining-reward-address" autocomplete="off" spellcheck="false" placeholder="VDR1…">
            </label>
            <div class="actions mining-actions">
              <button class="primary" id="start-mining" type="button">Start mining</button>
              <button class="danger" id="stop-mining" type="button">Stop mining</button>
            </div>
            <p class="warning">Testnet/Devnet mining only. Mainnet is disabled. Mining uses the local node RPC and does not expose private keys.</p>
            <p id="mining-message" class="subtle"></p>
          </article>
          <article class="card">
            <h3>Mining status</h3>
            <dl class="details compact">
              <div><dt>Process</dt><dd id="mining-state">Stopped</dd></div>
              <div><dt>Reward address</dt><dd><code id="mining-reward-current">—</code></dd></div>
              <div><dt>Accepted blocks</dt><dd id="mining-accepted">0</dd></div>
              <div><dt>Chain height</dt><dd id="mining-height">—</dd></div>
              <div><dt>Next height</dt><dd id="mining-next-height">—</dd></div>
              <div><dt>Block reward</dt><dd id="mining-reward">—</dd></div>
              <div><dt>Target block interval</dt><dd id="mining-target-time">—</dd></div>
              <div><dt>Current chain target</dt><dd><code id="mining-current-target">—</code></dd></div>
              <div><dt>Difficulty retarget</dt><dd id="mining-retarget">—</dd></div>
              <div><dt>Engine</dt><dd>CPU · SHA-256 · single thread</dd></div>
              <div><dt>Average effective hashrate</dt><dd id="mining-hashrate">—</dd></div>
              <div><dt>Last block hashrate</dt><dd id="mining-last-hashrate">—</dd></div>
              <div><dt>Last block solve time</dt><dd id="mining-last-duration">—</dd></div>
              <div><dt>Hashes tried · last block</dt><dd id="mining-last-hashes">—</dd></div>
              <div><dt>Hashes tried · this run</dt><dd id="mining-total-hashes">—</dd></div>
            </dl>
          </article>
        </div>
      </section>

      <section class="view" id="view-settings">
        <div class="section-head">
          <div>
            <p class="eyebrow" id="settings-eyebrow">DESKTOP PREFERENCES</p>
            <h2 id="settings-title">Settings</h2>
            <p class="subtle" id="settings-intro">Only non-secret convenience settings are stored locally.</p>
          </div>
        </div>
        <div class="settings-grid">
          <article class="card">
            <h3 id="settings-application-title">Application</h3>
            <form id="desktop-settings-form">
              <label>
                <span id="settings-language-label">Language</span>
                <select id="settings-language">
                  <option value="en">English</option>
                  <option value="ru">Русский</option>
                </select>
              </label>
              <label>
                <span id="settings-theme-label">Theme</span>
                <select id="settings-theme">
                  <option value="dark">Premium · reference</option>
                  <option value="classic">Classic dark</option>
                </select>
              </label>
              <label>
                <span id="settings-network-label">Network</span>
                <div class="settings-locked-field">
                  <strong>Testnet · valdr-testnet-1</strong>
                  <small id="settings-network-note">Mainnet is disabled in this build</small>
                </div>
              </label>
              <label class="settings-check">
                <input id="settings-start-node" type="checkbox">
                <span id="settings-start-node-label">Start managed local node when VALDR Desktop launches</span>
              </label>
              <label class="settings-check">
                <input id="settings-advanced" type="checkbox">
                <span id="settings-advanced-label">Enable Advanced mode</span>
              </label>
              <button class="primary" id="settings-save" type="submit">Save settings</button>
            </form>
            <p id="settings-status" class="subtle"></p>
          </article>

          <article class="card">
            <h3 id="settings-local-data-title">Local data</h3>
            <dl class="details">
              <div><dt>Application data</dt><dd><code id="settings-root">—</code></dd></div>
              <div><dt>Node data</dt><dd><code id="settings-node-data">—</code></dd></div>
              <div><dt>Wallets</dt><dd><code id="settings-wallets">—</code></dd></div>
              <div><dt>Logs</dt><dd><code id="settings-logs">—</code></dd></div>
            </dl>
            <p class="subtle" id="settings-data-note">Node data can be chosen during first run. After wallet creation, changing it remains disabled until a safe managed-node migration flow is implemented.</p>
          </article>

          <article class="card advanced-only hidden">
            <h3>Diagnostics export</h3>
            <p class="subtle">Save a local JSON report with node, storage, mining and redacted log diagnostics.</p>
            <p class="warning">Wallet private keys, passphrases and wallet file contents are not included.</p>
            <button class="secondary" id="export-diagnostics" type="button">Export diagnostics</button>
            <p id="diagnostics-export-status" class="subtle"></p>
          </article>
        </div>
      </section>
    </main>

    <aside class="quick-access-rail" aria-label="VALDR quick access">
      <button
        class="download-access"
        type="button"
        disabled
        title="Signed VALDR Desktop downloads will be enabled after the Stage 13 release gate"
        aria-label="VALDR downloads — pending signed release artifacts"
      >
        <span class="download-coin" aria-hidden="true">
          <img src="/valdr-emblem.svg" alt="">
          <span class="download-badge">↓</span>
        </span>
        <span class="download-caption">Downloads</span>
      </button>
    </aside>
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

const formatHashrate = (hashesPerSecond: number): string => {
  if (!Number.isFinite(hashesPerSecond) || hashesPerSecond <= 0) return "—";
  const units = ["H/s", "kH/s", "MH/s", "GH/s", "TH/s", "PH/s"];
  let value = hashesPerSecond;
  let unit = 0;
  while (value >= 1000 && unit < units.length - 1) {
    value /= 1000;
    unit++;
  }
  const digits = value >= 100 ? 0 : value >= 10 ? 1 : 2;
  return `${value.toFixed(digits)} ${units[unit]}`;
};

const formatMiningDuration = (durationMS: number): string => {
  if (!Number.isFinite(durationMS) || durationMS <= 0) return "—";
  if (durationMS < 1) return `${(durationMS * 1000).toFixed(0)} µs`;
  if (durationMS < 1000) {
    return `${durationMS.toFixed(durationMS >= 100 ? 0 : 1)} ms`;
  }
  return `${(durationMS / 1000).toFixed(2)} s`;
};

const formatHashCount = (count: number): string => {
  if (!Number.isFinite(count) || count <= 0) return "—";
  return Math.trunc(count).toLocaleString();
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

const sendError = document.getElementById("send-error");
const showSendError = (message: string): void => {
  if (!sendError) return;
  sendError.textContent = message;
  sendError.classList.remove("hidden");
};
const closeSendConfirmation = (): void => {
  document.getElementById("send-confirm-dialog")?.classList.add("hidden");
};

const closeTransactionDetail = (): void => {
  document.getElementById("transaction-detail-dialog")?.classList.add("hidden");
};

const openTransactionDetail = (item: TransactionHistoryItem): void => {
  const direction =
    item.direction === "received"
      ? "Received"
      : item.direction === "sent"
        ? "Sent"
        : "Self transfer";
  const prefix = item.direction === "received" ? "+" : item.direction === "sent" ? "−" : "";

  text("transaction-detail-status", item.status);
  text("transaction-detail-direction", direction);
  text("transaction-detail-time", new Date(item.timestamp * 1000).toLocaleString());
  text("transaction-detail-amount", `${prefix}${item.amount_vdr} VDR`);
  text("transaction-detail-fee", item.fee_val > 0 ? `${item.fee_vdr} VDR` : "—");
  text(
    "transaction-detail-confirmations",
    item.status === "pending" ? "0" : String(item.confirmations),
  );
  text(
    "transaction-detail-height",
    item.block_height === undefined ? "—" : String(item.block_height),
  );
  text("transaction-detail-block", item.block_hash || "—");
  text("transaction-detail-txid", item.transaction_id);
  document.getElementById("transaction-detail-dialog")?.classList.remove("hidden");
};

const clearSendStatus = (): void => {
  sendError?.classList.add("hidden");
  const result = document.getElementById("send-result");
  result?.classList.add("hidden");
  if (result) result.textContent = "";
  closeSendConfirmation();
};

let currentState: DesktopState | null = null;
let currentView = "overview";
let activeWalletAddress = "";
let activeWalletName = "";
let renderedReceiveQRAddress = "";
let lastPreviewInput: {
  selector: string;
  recipient: string;
  amount: string;
} | null = null;

let lastPreviewSummary: {
  amount: string;
  fee: string;
  total: string;
} | null = null;

const activeWallet = (): WalletMetadata | undefined =>
  currentState?.wallets.find((wallet) => wallet.address === activeWalletAddress);

const walletIsUnlocked = (): boolean =>
  Boolean(
    activeWalletAddress &&
      currentState?.unlocked_wallets.includes(activeWalletAddress),
  );

const clearPrivateKeyExport = (): void => {
  text("private-key-value", "");
  const confirmation = document.getElementById("export-confirmation") as HTMLInputElement | null;
  if (confirmation) confirmation.value = "";
  document.getElementById("export-warning")?.classList.add("hidden");
  document.getElementById("private-key-result")?.classList.add("hidden");
};

const applyDesktopTheme = (theme: DesktopPreferences["theme"]): void => {
  document.documentElement.dataset.theme = theme;
};

const applyDesktopLanguage = (language: DesktopPreferences["language"]): void => {
  document.documentElement.lang = language;

  const labels = language === "ru"
    ? {
        overview: "Главная",
        wallet: "Кошелёк",
        send: "Отправить",
        receive: "Получить",
        mining: "Майнинг",
        network: "Сеть",
        transactions: "Транзакции",
        settings: "Настройки",
        settingsEyebrow: "НАСТРОЙКИ DESKTOP",
        settingsTitle: "Настройки",
        settingsIntro: "Локально сохраняются только несекретные настройки приложения.",
        application: "Приложение",
        language: "Язык",
        theme: "Тема",
        networkLabel: "Сеть",
        networkNote: "Mainnet отключён в этой сборке",
        startNode: "Запускать локальную ноду при старте VALDR Desktop",
        advanced: "Включить расширенный режим",
        save: "Сохранить настройки",
        localData: "Локальные данные",
        dataNote: "Каталог данных ноды можно выбрать при первом запуске. После создания кошелька изменение отключено до реализации безопасного переноса управляемой ноды.",
        mainnet: "Mainnet отключён",
      }
    : {
        overview: "Dashboard",
        wallet: "Wallet",
        send: "Send",
        receive: "Receive",
        mining: "Mining",
        network: "Network",
        transactions: "Transactions",
        settings: "Settings",
        settingsEyebrow: "DESKTOP PREFERENCES",
        settingsTitle: "Settings",
        settingsIntro: "Only non-secret convenience settings are stored locally.",
        application: "Application",
        language: "Language",
        theme: "Theme",
        networkLabel: "Network",
        networkNote: "Mainnet is disabled in this build",
        startNode: "Start managed local node when VALDR Desktop launches",
        advanced: "Enable Advanced mode",
        save: "Save settings",
        localData: "Local data",
        dataNote: "Node data can be chosen during first run. After wallet creation, changing it remains disabled until a safe managed-node migration flow is implemented.",
        mainnet: "Mainnet disabled",
      };

  const navLabels: Record<string, string> = {
    overview: labels.overview,
    wallet: labels.wallet,
    send: labels.send,
    receive: labels.receive,
    mining: labels.mining,
    network: labels.network,
    transactions: labels.transactions,
    settings: labels.settings,
  };
  Object.entries(navLabels).forEach(([view, label]) => {
    const button = document.querySelector<HTMLButtonElement>(`.nav-item[data-view="${view}"]`);
    if (button) button.textContent = label;
  });

  text("settings-eyebrow", labels.settingsEyebrow);
  text("settings-title", labels.settingsTitle);
  text("settings-intro", labels.settingsIntro);
  text("settings-application-title", labels.application);
  text("settings-language-label", labels.language);
  text("settings-theme-label", labels.theme);
  text("settings-network-label", labels.networkLabel);
  text("settings-network-note", labels.networkNote);
  text("settings-start-node-label", labels.startNode);
  text("settings-advanced-label", labels.advanced);
  text("settings-save", labels.save);
  text("settings-local-data-title", labels.localData);
  text("settings-data-note", labels.dataNote);
  text("mainnet-status", labels.mainnet);

  const currentNav = document.querySelector<HTMLButtonElement>(`.nav-item[data-view="${currentView}"]`);
  if (currentNav) text("view-title", currentNav.textContent?.trim() || "VALDR");
};

const renderDesktopPreferences = (): void => {
  const prefs = currentState?.preferences;
  if (!prefs) return;

  const language = document.getElementById("settings-language") as HTMLSelectElement | null;
  const theme = document.getElementById("settings-theme") as HTMLSelectElement | null;
  const startNode = document.getElementById("settings-start-node") as HTMLInputElement | null;
  const advanced = document.getElementById("settings-advanced") as HTMLInputElement | null;
  if (language) language.value = prefs.language;
  if (theme) theme.value = prefs.theme;
  if (startNode) startNode.checked = prefs.start_node;
  if (advanced) advanced.checked = prefs.advanced;

  const publicNode = document.getElementById("public-node-enabled") as HTMLInputElement | null;
  const publicAddress = document.getElementById("public-node-address") as HTMLInputElement | null;
  if (publicNode) publicNode.checked = prefs.public_node;
  if (publicAddress) {
    publicAddress.value = prefs.public_node_advertise_address || "";
    publicAddress.disabled = !prefs.public_node;
  }

  applyDesktopTheme(prefs.theme);
  applyDesktopLanguage(prefs.language);

  document.querySelectorAll<HTMLElement>(".advanced-only").forEach((element) => {
    element.classList.toggle("hidden", !prefs.advanced);
  });

  if (!prefs.advanced && (currentView === "network" || currentView === "mining")) {
    const overview = document.querySelector<HTMLButtonElement>('.nav-item[data-view="overview"]');
    overview?.click();
  }
};

const renderWalletSecurity = (): void => {
  const wallet = activeWallet();
  const unlocked = Boolean(wallet) && walletIsUnlocked();
  text("wallet-lock-state", unlocked ? "Unlocked" : "Locked");
  text("send-wallet-lock-state", unlocked ? "Unlocked" : "Locked");

  const unlockForm = document.getElementById("unlock-wallet-form");
  unlockForm?.classList.toggle("hidden", !wallet || unlocked);

  const lockButton = document.getElementById("lock-wallet") as HTMLButtonElement | null;
  if (lockButton) lockButton.disabled = !wallet || !unlocked;

  const exportButton = document.getElementById("show-export-warning") as HTMLButtonElement | null;
  if (exportButton) exportButton.disabled = !wallet || !unlocked;
  if (!wallet || !unlocked) clearPrivateKeyExport();

  const autoLock = document.getElementById("wallet-auto-lock") as HTMLSelectElement | null;
  if (autoLock && currentState?.wallet_auto_lock_minutes) {
    autoLock.value = String(currentState.wallet_auto_lock_minutes);
  }
};

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
      lastPreviewSummary = null;
      renderedReceiveQRAddress = "";
      clearPrivateKeyExport();
      closeSendConfirmation();
      void refreshWalletPresentation();
      renderWalletSecurity();
      renderWallets(wallets);
    });
    list.append(item);
  }
};

const refreshReceiveQR = async (address: string): Promise<void> => {
  const panel = document.getElementById("receive-qr-panel");
  const image = document.getElementById("receive-qr") as HTMLImageElement | null;

  if (!address) {
    renderedReceiveQRAddress = "";
    if (image) image.removeAttribute("src");
    panel?.classList.add("hidden");
    return;
  }
  if (address === renderedReceiveQRAddress && image?.src) {
    panel?.classList.remove("hidden");
    return;
  }

  try {
    const dataURI = await api().GetReceiveQRCode(address);
    if (activeWalletAddress !== address) return;
    if (image) image.src = dataURI;
    renderedReceiveQRAddress = address;
    panel?.classList.remove("hidden");
  } catch {
    renderedReceiveQRAddress = "";
    if (image) image.removeAttribute("src");
    panel?.classList.add("hidden");
  }
};

const refreshWalletPresentation = async (): Promise<void> => {
  const wallet = activeWallet();
  text("send-from", wallet ? wallet.address : "");
  const sendFrom = document.getElementById("send-from") as HTMLInputElement | null;
  if (sendFrom) sendFrom.value = wallet?.address ?? "";
  text("receive-address", wallet?.address ?? "Select a wallet");
  void refreshReceiveQR(wallet?.address ?? "");
  text(
    "overview-wallet-label",
    wallet
      ? `${activeWalletName} · ${walletIsUnlocked() ? "Unlocked" : "Locked"}`
      : "Create or select an encrypted wallet.",
  );

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
  if (currentView === "overview") {
    void refreshOverviewLatestTransaction();
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

    const actions = document.createElement("div");
    actions.className = "history-actions";
    const detailButton = document.createElement("button");
    detailButton.type = "button";
    detailButton.className = "secondary";
    detailButton.textContent = "View details";
    detailButton.addEventListener("click", () => openTransactionDetail(item));
    actions.append(detailButton);

    card.append(head, meta, actions);
    list.append(card);
  }
};

const refreshOverviewLatestTransaction = async (): Promise<void> => {
  const wallet = activeWallet();
  if (!wallet || !currentState?.node_status) {
    text("overview-latest-direction", "No transactions yet");
    text("overview-latest-amount", "—");
    text(
      "overview-latest-meta",
      wallet ? "Waiting for local node status." : "Select a wallet to view its latest activity.",
    );
    return;
  }

  try {
    const items = await api().GetTransactionHistory(wallet.address);
    if (activeWalletAddress !== wallet.address) return;
    const item = items[0];
    if (!item) {
      text("overview-latest-direction", "No transactions yet");
      text("overview-latest-amount", "—");
      text("overview-latest-meta", "No wallet activity found.");
      return;
    }

    const direction =
      item.direction === "received"
        ? "Received"
        : item.direction === "sent"
          ? "Sent"
          : "Self transfer";
    const prefix = item.direction === "received" ? "+" : item.direction === "sent" ? "−" : "";
    text("overview-latest-direction", direction);
    text("overview-latest-amount", `${prefix}${item.amount_vdr} VDR`);
    text(
      "overview-latest-meta",
      `${item.status} · ${new Date(item.timestamp * 1000).toLocaleString()} · ${item.transaction_id.slice(0, 16)}…`,
    );
  } catch {
    text("overview-latest-direction", "Unavailable");
    text("overview-latest-amount", "—");
    text("overview-latest-meta", "Latest transaction could not be loaded.");
  }
};

const renderPeers = (peers: PeerInfo[]): void => {
  const list = document.getElementById("peer-list");
  const empty = document.getElementById("peer-list-empty");
  if (!list || !empty) return;
  list.replaceChildren();

  if (peers.length === 0) {
    empty.classList.remove("hidden");
    return;
  }
  empty.classList.add("hidden");

  for (const peer of peers) {
    const item = document.createElement("div");
    item.className = "peer-item";

    const identity = document.createElement("div");
    const nodeID = document.createElement("strong");
    nodeID.textContent = peer.node_id || "Unknown peer";
    const address = document.createElement("code");
    address.textContent = peer.address;
    identity.append(nodeID, address);

    const meta = document.createElement("div");
    meta.className = "peer-meta";
    const direction = peer.inbound ? "Inbound" : "Outbound";
    meta.textContent = `${direction} · v${peer.protocol_version} · height ${peer.height}`;

    item.append(identity, meta);
    list.append(item);
  }
};

const refreshPeers = async (): Promise<void> => {
  if (!currentState?.node_running) {
    renderPeers([]);
    return;
  }
  try {
    renderPeers(await api().GetPeers());
  } catch {
    renderPeers([]);
  }
};

const refreshNodeLogs = async (): Promise<void> => {
  const output = document.getElementById("node-log-output") as HTMLPreElement | null;
  if (!output) return;
  if (!currentState?.preferences.advanced) {
    output.textContent = "Advanced mode is required for node diagnostics.";
    return;
  }
  try {
    const logs = (await api().GetNodeLogs()).trim();
    output.textContent = logs || "No node log output yet.";
    output.scrollTop = output.scrollHeight;
  } catch (error) {
    output.textContent = error instanceof Error ? error.message : String(error);
  }
};

const formatBytes = (value: number): string => {
  if (!Number.isFinite(value) || value < 0) return "—";
  if (value < 1024) return `${Math.round(value)} B`;
  const units = ["KiB", "MiB", "GiB", "TiB"];
  let amount = value / 1024;
  let unit = units[0];
  for (let i = 1; i < units.length && amount >= 1024; i++) {
    amount /= 1024;
    unit = units[i];
  }
  return `${amount >= 100 ? amount.toFixed(0) : amount >= 10 ? amount.toFixed(1) : amount.toFixed(2)} ${unit}`;
};

const refreshStorageDiagnostics = async (): Promise<void> => {
  if (!currentState?.preferences.advanced) return;
  text("storage-diagnostics-status", "Inspecting local node data…");
  try {
    const diagnostics = await api().GetStorageDiagnostics();
    text(
      "detail-storage-state",
      diagnostics.ready ? (diagnostics.truncated ? "Ready · bounded scan" : "Ready") : "Unavailable",
    );
    text("detail-storage-size", formatBytes(diagnostics.size_bytes));
    text(
      "detail-storage-files",
      diagnostics.truncated ? `${diagnostics.file_count}+` : String(diagnostics.file_count),
    );
    text(
      "storage-diagnostics-status",
      diagnostics.error ||
        (diagnostics.truncated
          ? "Directory is healthy. Scan stopped at the safety entry limit."
          : "Directory is readable and the bounded inspection completed."),
    );
  } catch (error) {
    text("detail-storage-state", "Unavailable");
    text("detail-storage-size", "—");
    text("detail-storage-files", "—");
    text("storage-diagnostics-status", error instanceof Error ? error.message : String(error));
  }
};

const refreshExplorerStatus = async (): Promise<void> => {
  const openButton = document.getElementById("open-explorer") as HTMLButtonElement | null;
  if (openButton) openButton.disabled = true;
  if (!currentState?.preferences.advanced) return;

  text("explorer-status", "Checking local Explorer…");
  try {
    const status = await api().GetExplorerStatus();
    if (openButton) openButton.disabled = !status.available;
    text(
      "explorer-status",
      status.available
        ? `Ready · Testnet height ${status.height ?? 0} · ${status.url}`
        : (status.message || "Local Explorer is not running."),
    );
  } catch (error) {
    text("explorer-status", error instanceof Error ? error.message : String(error));
  }
};

const refreshMining = async (): Promise<void> => {
  const rewardInput = document.getElementById("mining-reward-address") as HTMLInputElement | null;
  if (rewardInput && !rewardInput.value && activeWalletAddress) {
    rewardInput.value = activeWalletAddress;
  }

  const startButton = document.getElementById("start-mining") as HTMLButtonElement | null;
  const stopButton = document.getElementById("stop-mining") as HTMLButtonElement | null;

  if (!currentState?.preferences.advanced) {
    if (startButton) startButton.disabled = true;
    if (stopButton) stopButton.disabled = true;
    return;
  }
  if (!currentState.node_running) {
    text("mining-state", "Node stopped");
    text("mining-message", "Start the managed local node before mining.");
    if (startButton) startButton.disabled = true;
    if (stopButton) stopButton.disabled = true;
    return;
  }

  try {
    const status = await api().GetMiningState();
    text("mining-state", status.running ? "Running" : "Stopped");
    text("mining-reward-current", status.reward_address || "—");
    text("mining-accepted", String(status.accepted_blocks));
    text("mining-height", String(status.height));
    text("mining-next-height", String(status.next_height));
    text("mining-reward", status.block_reward_vdr ? status.block_reward_vdr + " VDR" : "—");
    text(
      "mining-target-time",
      status.target_block_time_seconds > 0
        ? status.target_block_time_seconds + " seconds"
        : "—",
    );
    text("mining-current-target", status.current_target || "—");
    text(
      "mining-retarget",
      status.retarget_interval > 0
        ? status.blocks_until_retarget === 0
          ? `Next block · every ${status.retarget_interval} blocks`
          : `In ${status.blocks_until_retarget} block(s) · every ${status.retarget_interval} blocks`
        : "—",
    );
    text("mining-hashrate", formatHashrate(status.hashrate_hps));
    text("mining-last-hashrate", formatHashrate(status.last_block_hashrate_hps));
    text("mining-last-duration", formatMiningDuration(status.last_block_duration_ms));
    text("mining-last-hashes", formatHashCount(status.last_block_hashes));
    text("mining-total-hashes", formatHashCount(status.total_hashes));
    text("mining-message", status.last_error || "");
    if (startButton) startButton.disabled = status.running;
    if (stopButton) stopButton.disabled = !status.running;
  } catch (error) {
    text("mining-state", "Unavailable");
    text("mining-message", error instanceof Error ? error.message : String(error));
    if (startButton) startButton.disabled = true;
    if (stopButton) stopButton.disabled = true;
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
  text("detail-logs", state.paths.logs);
  text("settings-root", state.paths.root);
  text("settings-node-data", state.paths.node_data);
  text("settings-wallets", state.paths.wallets);
  text("settings-logs", state.paths.logs);
  text("first-run-data-path", state.paths.node_data);
  renderDesktopPreferences();

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
  text("top-block", status ? String(status.height) : "—");
  text("top-peers", status ? String(status.peer_count) : "—");
  text("detail-tip", status?.tip_hash || "—");
  text("detail-chainwork", status?.chainwork || "—");
  text("detail-peer-count", status ? String(status.peer_count) : "—");
  text(
    "detail-height",
    status ? `${status.height} / ${status.best_known_height}` : "—",
  );
  text(
    "detail-sync",
    status ? `${Math.round(status.sync_progress * 100)}%` : "—",
  );

  const healthy = state.node_running && Boolean(status);
  text(
    "detail-node-state",
    healthy ? "Online" : state.node_running ? "Starting" : "Stopped",
  );
  text("node-badge-text", healthy ? "Node online" : state.node_running ? "Node starting" : "Node stopped");
  text("node-state", healthy ? "Connected to local RPC" : state.node_running ? "Starting local node…" : "Stopped");

  const badge = document.getElementById("node-badge");
  badge?.classList.toggle("healthy", healthy);

  const startNodeButton = document.getElementById("start-node") as HTMLButtonElement | null;
  const stopNodeButton = document.getElementById("stop-node") as HTMLButtonElement | null;
  if (startNodeButton) startNodeButton.disabled = state.node_running;
  if (stopNodeButton) stopNodeButton.disabled = !state.node_running;
  const firstRunRestartVisible =
    state.wallets.length > 0 &&
    !state.initialization_ready &&
    (!state.node_running || Boolean(state.node_error));
  document.querySelectorAll<HTMLButtonElement>(".restart-node-action").forEach((button) => {
    const visible = button.id === "first-run-restart-node"
      ? firstRunRestartVisible
      : Boolean(state.node_error);
    button.classList.toggle("hidden", !visible);
    button.disabled = false;
  });

  const firstRun = document.getElementById("first-run");
  firstRun?.classList.toggle("hidden", state.initialization_ready);
  document.getElementById("first-run-wallet-setup")?.classList.toggle(
    "hidden",
    state.wallets.length > 0,
  );
  document.getElementById("first-run-initializing")?.classList.toggle(
    "hidden",
    state.wallets.length === 0 || state.initialization_ready,
  );
  const chooseData = document.getElementById("first-run-choose-data") as HTMLButtonElement | null;
  if (chooseData) chooseData.disabled = state.wallets.length > 0;
  text(
    "first-run-node-state",
    state.wallets.length === 0
      ? "Local node starts after wallet setup"
      : state.node_error
        ? "Local node needs attention"
        : healthy
          ? "Local validating node online"
          : state.node_running
            ? "Starting local validating node…"
            : "Local node stopped",
  );
  text(
    "first-run-sync",
    state.wallets.length === 0
      ? "Waiting for encrypted wallet"
      : !status
        ? "Waiting for node status"
        : status.peer_count === 0
          ? `Height ${status.height} · waiting for peers`
          : `Sync ${Math.round(status.sync_progress * 100)}% · ${status.height}/${status.best_known_height}`,
  );

  if (state.node_error) {
    text("node-detail", state.node_error);
  } else if (healthy && status) {
    const mode = state.preferences.public_node ? "Public P2P" : "Outbound-only";
    text("node-detail", `${mode} · ${status.peer_count} peer(s) · height ${status.height}`);
  } else {
    text(
      "node-detail",
      state.preferences.public_node
        ? "Public Testnet P2P mode configured."
        : "Desktop uses outbound-only P2P by default.",
    );
  }

  renderWalletSelector(state.wallets);
  renderWallets(state.wallets);
  renderWalletSecurity();
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

const navigateToView = (view: string): void => {
  const navButton = document.querySelector<HTMLButtonElement>(`.nav-item[data-view="${view}"]`);
  if (!navButton || navButton.classList.contains("hidden")) return;
  if (view !== "wallet") clearPrivateKeyExport();
  currentView = view;
  closeTransactionDetail();
  document.querySelectorAll(".nav-item").forEach((item) => item.classList.remove("active"));
  navButton.classList.add("active");
  document.querySelectorAll(".view").forEach((item) => item.classList.remove("active"));
  document.getElementById(`view-${view}`)?.classList.add("active");
  text("view-title", navButton.textContent?.trim() || "VALDR");
  if (view === "transactions") {
    void refreshTransactionHistory();
  } else if (view === "overview") {
    void refreshOverviewLatestTransaction();
  } else if (view === "network") {
    void refreshPeers();
    void refreshNodeLogs();
    void refreshStorageDiagnostics();
    void refreshExplorerStatus();
  } else if (view === "mining") {
    void refreshMining();
  }
};

document.querySelectorAll<HTMLButtonElement>(".nav-item[data-view]").forEach((button) => {
  button.addEventListener("click", () => {
    const view = button.dataset.view;
    if (view) navigateToView(view);
  });
});

document.querySelectorAll<HTMLButtonElement>(".view-shortcut[data-go-view]").forEach((button) => {
  button.addEventListener("click", () => {
    const view = button.dataset.goView;
    if (view) navigateToView(view);
  });
});

document.getElementById("wallet-selector")?.addEventListener("change", (event) => {
  const selector = event.currentTarget as HTMLSelectElement;
  activeWalletAddress = selector.value;
  activeWalletName = activeWallet()?.name || "VALDR Wallet";
  renderedReceiveQRAddress = "";
  lastPreviewInput = null;
  lastPreviewSummary = null;
  clearPrivateKeyExport();
  closeSendConfirmation();
  closeTransactionDetail();
  clearSendStatus();
  document.getElementById("send-preview")?.classList.add("hidden");
  document.getElementById("send-preview-empty")?.classList.remove("hidden");
  if (currentState) renderWallets(currentState.wallets);
  renderWalletSecurity();
  void refreshWalletPresentation();
  if (currentView === "transactions") {
    void refreshTransactionHistory();
  }
});

const saveDesktopSettings = async (): Promise<void> => {
  const language = ((document.getElementById("settings-language") as HTMLSelectElement | null)?.value ?? "en") as DesktopPreferences["language"];
  const theme = ((document.getElementById("settings-theme") as HTMLSelectElement | null)?.value ?? "dark") as DesktopPreferences["theme"];
  const startNode = (document.getElementById("settings-start-node") as HTMLInputElement | null)?.checked ?? true;
  const advanced = (document.getElementById("settings-advanced") as HTMLInputElement | null)?.checked ?? false;

  // Apply the visible choices immediately; backend persistence is authoritative.
  applyDesktopTheme(theme);
  applyDesktopLanguage(language);
  document.querySelectorAll<HTMLElement>(".advanced-only").forEach((element) => {
    element.classList.toggle("hidden", !advanced);
  });

  try {
    const preferences = await api().SetDesktopPreferences(language, theme, startNode, advanced);
    if (currentState) currentState.preferences = preferences;
    renderDesktopPreferences();
    text(
      "settings-status",
      preferences.language === "ru"
        ? "Настройки сохранены. Параметр запуска ноды применяется при следующем запуске приложения."
        : "Settings saved. Node startup behavior applies on the next application launch.",
    );
  } catch (error) {
    if (currentState) renderDesktopPreferences();
    text("settings-status", error instanceof Error ? error.message : String(error));
  }
};

document.getElementById("export-diagnostics")?.addEventListener("click", async () => {
  const button = document.getElementById("export-diagnostics") as HTMLButtonElement | null;
  if (button) button.disabled = true;
  text("diagnostics-export-status", "Preparing diagnostics…");
  try {
    const path = await api().ExportDiagnostics();
    text(
      "diagnostics-export-status",
      path ? `Diagnostics saved: ${path}` : "Diagnostics export cancelled.",
    );
  } catch (error) {
    text(
      "diagnostics-export-status",
      error instanceof Error ? error.message : String(error),
    );
  } finally {
    if (button) button.disabled = false;
  }
});

document.getElementById("desktop-settings-form")?.addEventListener("submit", (event) => {
  event.preventDefault();
  void saveDesktopSettings();
});

["settings-language", "settings-theme", "settings-start-node", "settings-advanced"].forEach((id) => {
  document.getElementById(id)?.addEventListener("change", () => {
    void saveDesktopSettings();
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

document.querySelectorAll<HTMLButtonElement>(".restart-node-action").forEach((button) => {
  button.addEventListener("click", async () => {
    clearError();
    button.disabled = true;
    text("node-detail", "Restarting local node…");
    text("first-run-node-state", "Restarting local node…");
    try {
      await api().RestartNode();
      await refresh();
    } catch (error) {
      showError(error instanceof Error ? error.message : String(error));
    } finally {
      button.disabled = false;
    }
  });
});

document.getElementById("start-mining")?.addEventListener("click", async () => {
  const reward = value("mining-reward-address").trim() || activeWalletAddress;
  if (!reward) {
    text("mining-message", "Select a wallet or enter a VALDR reward address.");
    return;
  }
  try {
    text("mining-message", "");
    await api().StartMining(reward);
    await refreshMining();
  } catch (error) {
    text("mining-message", error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("stop-mining")?.addEventListener("click", async () => {
  try {
    text("mining-message", "");
    await api().StopMining();
    await refreshMining();
  } catch (error) {
    text("mining-message", error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("first-run-choose-data")?.addEventListener("click", async () => {
  const button = document.getElementById("first-run-choose-data") as HTMLButtonElement | null;
  if (button) button.disabled = true;
  text("first-run-error", "");
  try {
    const path = await api().ChooseFirstRunNodeDataDirectory();
    text("first-run-data-path", path);
    await refresh();
  } catch (error) {
    text("first-run-error", error instanceof Error ? error.message : String(error));
    await refresh();
  } finally {
    if (button && (currentState?.wallets.length ?? 0) === 0) {
      button.disabled = false;
    }
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
    await refresh();
  } catch (error) {
    text("first-run-error", error instanceof Error ? error.message : String(error));
  } finally {
    const passInput = document.getElementById("first-wallet-passphrase") as HTMLInputElement | null;
    const confirmInput = document.getElementById("first-wallet-confirm") as HTMLInputElement | null;
    if (passInput) passInput.value = "";
    if (confirmInput) confirmInput.value = "";
  }
});

document.getElementById("first-run-restore")?.addEventListener("click", async () => {
  const input = document.getElementById("first-run-restore-passphrase") as HTMLInputElement | null;
  const passphrase = input?.value ?? "";
  if (!passphrase) {
    text("first-run-error", "Backup passphrase is required.");
    return;
  }
  if (input) input.value = "";

  try {
    text("first-run-error", "");
    const restored = await api().RestoreWallet(passphrase);
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
    await refresh();
  } catch (error) {
    showError(error instanceof Error ? error.message : String(error));
  } finally {
    const passInput = document.getElementById("wallet-passphrase") as HTMLInputElement | null;
    const confirmInput = document.getElementById("wallet-passphrase-confirm") as HTMLInputElement | null;
    if (passInput) passInput.value = "";
    if (confirmInput) confirmInput.value = "";
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

document.getElementById("restore-wallet-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  const input = document.getElementById("restore-wallet-passphrase") as HTMLInputElement | null;
  const passphrase = input?.value ?? "";
  if (!passphrase) {
    text("wallet-action-status", "Backup passphrase is required.");
    return;
  }
  if (input) input.value = "";

  try {
    const restored = await api().RestoreWallet(passphrase);
    if (!restored.address) {
      text("wallet-action-status", "Restore cancelled.");
      return;
    }
    activeWalletAddress = restored.address;
    activeWalletName = restored.name || "VALDR Wallet";
    text("wallet-action-status", "Encrypted wallet restored and unlocked locally.");
    await refresh();
  } catch (error) {
    text(
      "wallet-action-status",
      error instanceof Error ? error.message : String(error),
    );
  }
});

document.getElementById("unlock-wallet-form")?.addEventListener("submit", async (event) => {
  event.preventDefault();
  const wallet = activeWallet();
  if (!wallet) {
    text("wallet-security-status", "Select a wallet first.");
    return;
  }
  const passphrase = value("unlock-wallet-passphrase");
  if (!passphrase) {
    text("wallet-security-status", "Wallet passphrase is required.");
    return;
  }
  try {
    await api().UnlockWallet(wallet.address, passphrase);
    text("wallet-security-status", "Wallet unlocked locally.");
    await refresh();
  } catch (error) {
    text(
      "wallet-security-status",
      error instanceof Error ? error.message : String(error),
    );
  } finally {
    const input = document.getElementById("unlock-wallet-passphrase") as HTMLInputElement | null;
    if (input) input.value = "";
  }
});

document.getElementById("lock-wallet")?.addEventListener("click", async () => {
  const wallet = activeWallet();
  if (!wallet) {
    text("wallet-security-status", "Select a wallet first.");
    return;
  }
  try {
    await api().LockWallet(wallet.address);
    lastPreviewInput = null;
    lastPreviewSummary = null;
    clearSendStatus();
    document.getElementById("send-preview")?.classList.add("hidden");
    document.getElementById("send-preview-empty")?.classList.remove("hidden");
    text("wallet-security-status", "Wallet locked.");
    await refresh();
  } catch (error) {
    text(
      "wallet-security-status",
      error instanceof Error ? error.message : String(error),
    );
  }
});

document.getElementById("show-export-warning")?.addEventListener("click", () => {
  clearPrivateKeyExport();
  document.getElementById("export-warning")?.classList.remove("hidden");
});

document.getElementById("cancel-private-key-export")?.addEventListener("click", () => {
  clearPrivateKeyExport();
});

document.getElementById("hide-private-key")?.addEventListener("click", () => {
  clearPrivateKeyExport();
});

document.getElementById("confirm-private-key-export")?.addEventListener("click", async () => {
  const wallet = activeWallet();
  if (!wallet || !walletIsUnlocked()) {
    text("wallet-security-status", "Unlock the selected wallet before exporting.");
    clearPrivateKeyExport();
    return;
  }
  const confirmation = value("export-confirmation").trim();
  try {
    const privateKey = await api().ExportPrivateKey(wallet.address, confirmation);
    text("private-key-value", privateKey);
    const input = document.getElementById("export-confirmation") as HTMLInputElement | null;
    if (input) input.value = "";
    document.getElementById("export-warning")?.classList.add("hidden");
    document.getElementById("private-key-result")?.classList.remove("hidden");
  } catch (error) {
    text(
      "wallet-security-status",
      error instanceof Error ? error.message : String(error),
    );
  }
});

document.getElementById("wallet-auto-lock")?.addEventListener("change", async (event) => {
  const select = event.currentTarget as HTMLSelectElement;
  const minutes = Number(select.value);
  try {
    await api().SetWalletAutoLockMinutes(minutes);
    text("wallet-security-status", `Auto-lock set to ${minutes} minute(s).`);
    await refresh();
  } catch (error) {
    text(
      "wallet-security-status",
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
  if (!walletIsUnlocked()) {
    showSendError("Unlock the selected wallet in the Wallet screen before signing.");
    return;
  }

  const recipient = value("send-recipient").trim();
  const amount = value("send-amount").trim();
  if (!recipient || !amount) {
    showSendError("Recipient and amount are required.");
    return;
  }

  try {
    const preview = await api().PreviewSend(
      wallet.address,
      recipient,
      amount,
    );
    lastPreviewInput = {
      selector: wallet.address,
      recipient,
      amount,
    };
    lastPreviewSummary = {
      amount: preview.amount_vdr,
      fee: preview.fee_vdr,
      total: preview.total_vdr,
    };
    text("preview-amount", preview.amount_vdr);
    text("preview-fee", preview.fee_vdr);
    text("preview-total", preview.total_vdr);
    document.getElementById("send-preview-empty")?.classList.add("hidden");
    document.getElementById("send-preview")?.classList.remove("hidden");
  } catch (error) {
    lastPreviewInput = null;
    lastPreviewSummary = null;
    showSendError(error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("confirm-send")?.addEventListener("click", () => {
  clearSendStatus();
  const input = lastPreviewInput;
  const summary = lastPreviewSummary;
  if (!input || !summary) {
    showSendError("Preview the transaction again before broadcasting.");
    return;
  }
  if (!walletIsUnlocked()) {
    showSendError("The wallet is locked. Unlock it and preview the transaction again.");
    lastPreviewInput = null;
    document.getElementById("send-preview")?.classList.add("hidden");
    document.getElementById("send-preview-empty")?.classList.remove("hidden");
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

  text("confirm-recipient", input.recipient);
  text("confirm-amount", summary.amount);
  text("confirm-fee", summary.fee);
  text("confirm-total", summary.total);
  document.getElementById("send-confirm-dialog")?.classList.remove("hidden");
});

document.getElementById("cancel-send-confirmation")?.addEventListener("click", () => {
  closeSendConfirmation();
});

document.getElementById("broadcast-send")?.addEventListener("click", async () => {
  const input = lastPreviewInput;
  if (!input || !lastPreviewSummary) {
    closeSendConfirmation();
    showSendError("Preview the transaction again before broadcasting.");
    return;
  }
  if (!walletIsUnlocked()) {
    closeSendConfirmation();
    lastPreviewInput = null;
    lastPreviewSummary = null;
    document.getElementById("send-preview")?.classList.add("hidden");
    document.getElementById("send-preview-empty")?.classList.remove("hidden");
    showSendError("The wallet is locked. Unlock it and preview the transaction again.");
    return;
  }

  const recipientNow = value("send-recipient").trim();
  const amountNow = value("send-amount").trim();
  if (recipientNow !== input.recipient || amountNow !== input.amount) {
    closeSendConfirmation();
    lastPreviewInput = null;
    lastPreviewSummary = null;
    document.getElementById("send-preview")?.classList.add("hidden");
    document.getElementById("send-preview-empty")?.classList.remove("hidden");
    showSendError("Transaction details changed. Preview the fee again.");
    return;
  }

  try {
    const result = await api().SendTransaction(
      input.selector,
      input.recipient,
      input.amount,
    );
    lastPreviewInput = null;
    lastPreviewSummary = null;
    closeSendConfirmation();

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
    closeSendConfirmation();
    showSendError(error instanceof Error ? error.message : String(error));
  }
});

document.getElementById("close-transaction-detail")?.addEventListener("click", () => {
  closeTransactionDetail();
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    closeSendConfirmation();
    closeTransactionDetail();
  }
});

document.addEventListener("visibilitychange", () => {
  if (document.hidden) {
    clearPrivateKeyExport();
    closeSendConfirmation();
  }
});

document.getElementById("refresh-history")?.addEventListener("click", async () => {
  await refreshTransactionHistory();
});

document.getElementById("public-node-enabled")?.addEventListener("change", (event) => {
  const enabled = (event.currentTarget as HTMLInputElement).checked;
  const address = document.getElementById("public-node-address") as HTMLInputElement | null;
  if (address) address.disabled = !enabled;
});

document.getElementById("apply-public-node")?.addEventListener("click", async () => {
  const enabled = (document.getElementById("public-node-enabled") as HTMLInputElement | null)?.checked ?? false;
  const addressInput = document.getElementById("public-node-address") as HTMLInputElement | null;
  const advertiseAddress = addressInput?.value.trim() ?? "";
  const button = document.getElementById("apply-public-node") as HTMLButtonElement | null;

  if (button) button.disabled = true;
  text("public-node-status", enabled ? "Applying public-node mode…" : "Applying outbound-only mode…");
  try {
    const preferences = await api().SetPublicNodeMode(enabled, advertiseAddress);
    if (currentState) currentState.preferences = preferences;
    await refresh();
    text(
      "public-node-status",
      preferences.public_node
        ? `Public Testnet P2P enabled · advertising ${preferences.public_node_advertise_address}`
        : "Outbound-only Desktop P2P enabled.",
    );
  } catch (error) {
    text("public-node-status", error instanceof Error ? error.message : String(error));
    if (currentState) renderDesktopPreferences();
  } finally {
    if (button) button.disabled = false;
  }
});

document.getElementById("refresh-peers")?.addEventListener("click", async () => {
  await refreshPeers();
});

document.getElementById("refresh-node-logs")?.addEventListener("click", async () => {
  await refreshNodeLogs();
});

document.getElementById("refresh-storage-diagnostics")?.addEventListener("click", async () => {
  await refreshStorageDiagnostics();
});

document.getElementById("refresh-explorer-status")?.addEventListener("click", async () => {
  await refreshExplorerStatus();
});

document.getElementById("open-explorer")?.addEventListener("click", async () => {
  const button = document.getElementById("open-explorer") as HTMLButtonElement | null;
  if (button) button.disabled = true;
  try {
    await api().OpenExplorer();
    text("explorer-status", "Opened local VALDR Explorer in your default browser.");
  } catch (error) {
    text("explorer-status", error instanceof Error ? error.message : String(error));
  } finally {
    await refreshExplorerStatus();
  }
});

document.getElementById("copy-address")?.addEventListener("click", async () => {
  const wallet = activeWallet();
  if (!wallet) {
    text("copy-status", "Select a wallet first.");
    return;
  }
  try {
    await api().CopyReceiveAddress(wallet.address);
    text("copy-status", "Address copied.");
  } catch (error) {
    text(
      "copy-status",
      error instanceof Error ? error.message : "Clipboard unavailable.",
    );
  }
});

window.setTimeout(() => {
  document.getElementById("boot-splash")?.classList.add("boot-splash-done");
  window.setTimeout(() => document.getElementById("boot-splash")?.remove(), 520);
}, 1250);

void refresh();
window.setInterval(() => {
  void refresh();
  if (currentView === "transactions") {
    void refreshTransactionHistory();
  } else if (currentView === "mining") {
    void refreshMining();
  }
}, 5000);
