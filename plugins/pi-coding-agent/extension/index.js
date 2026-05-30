import { randomBytes } from "node:crypto";
import { chmodSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createServer } from "node:net";
import { homedir } from "node:os";
import { join } from "node:path";

const BRIDGE_DIR = "ero-pi-coding-agent-bridge";
const STATE_FILE = "sessions.json";

export default function eroPiCodingAgentBridge(pi) {
  const token = randomBytes(32).toString("hex");
  const bridgeDir = bridgeRuntimeDir();
  const statePath = join(bridgeDir, STATE_FILE);
  let currentSessionId;
  let socketPath;

  const server = createServer((conn) => {
    let data = "";
    let closed = false;
    let rejected = false;

    function respond(response) {
      if (closed || conn.destroyed || conn.writableEnded) return;
      closed = true;
      conn.end(JSON.stringify(response) + "\n");
    }

    conn.setEncoding("utf8");
    conn.on("error", () => {
      closed = true;
    });
    conn.on("data", (chunk) => {
      if (rejected) return;
      data += chunk;
      if (data.length > 1024 * 1024) {
        rejected = true;
        respond({ ok: false, error: "request body too large" });
      }
    });
    conn.on("end", () => {
      if (rejected || closed) return;

      let payload;
      try {
        payload = JSON.parse(data);
      } catch (_error) {
        respond({ ok: false, error: "invalid json" });
        return;
      }

      if (!currentSessionId || payload.session_id !== currentSessionId || payload.token !== token) {
        respond({ ok: false, error: "invalid bridge token or session" });
        return;
      }
      if (!payload.message?.trim()) {
        respond({ ok: false, error: "message is required" });
        return;
      }

      try {
        const delivery = Promise.resolve(pi.sendUserMessage(payload.message, { deliverAs: "followUp" }));
        respond({ ok: true });
        delivery.catch((error) => {
          const message = error instanceof Error ? error.message : String(error);
          pi.ui?.notify?.(`Ero bridge failed to deliver review: ${message}`, "error");
        });
      } catch (error) {
        respond({ ok: false, error: error instanceof Error ? error.message : String(error) });
      }
    });
  });

  function ensureListening(sessionId) {
    const nextSocketPath = join(bridgeDir, `${sessionId}.sock`);
    if (socketPath === nextSocketPath && server.listening) return;
    if (server.listening) server.close();
    rmSync(nextSocketPath, { force: true });
    socketPath = nextSocketPath;
    server.listen(socketPath, () => {
      chmodSync(socketPath, 0o600);
    });
  }

  async function register(ctx) {
    currentSessionId = ctx.sessionManager.getSessionId();
    ensureListening(currentSessionId);
    const git = await readGitMetadata(pi, ctx.cwd);
    upsertSession(statePath, {
      session_id: currentSessionId,
      session_file: ctx.sessionManager.getSessionFile(),
      cwd: ctx.cwd,
      worktree_root: git.worktreeRoot,
      current_branch: git.currentBranch,
      head_sha: git.headSHA,
      socket_path: socketPath,
      token,
      updated_at: new Date().toISOString(),
    });
    ctx.ui.setStatus("ero-pi-coding-agent", `Ero bridge ${currentSessionId.slice(0, 8)}`);
  }

  pi.on("session_start", async (_event, ctx) => {
    await register(ctx);
  });

  pi.on("session_shutdown", async (_event, ctx) => {
    removeSession(statePath, ctx.sessionManager.getSessionId());
    if (server.listening) server.close();
    if (socketPath) rmSync(socketPath, { force: true });
    ctx.ui.setStatus("ero-pi-coding-agent", undefined);
  });

  pi.registerCommand("ero-bridge", {
    description: "Show the active Ero/pi-coding-agent bridge session ID",
    handler: async (_args, ctx) => {
      await register(ctx);
      ctx.ui.notify(
        `Ero pi-coding-agent bridge is active for session ${ctx.sessionManager.getSessionId()} at ${statePath}`,
        "info",
      );
    },
  });
}

async function readGitMetadata(pi, cwd) {
  const [worktreeRoot, currentBranch, headSHA] = await Promise.all([
    gitOutput(pi, cwd, ["rev-parse", "--show-toplevel"]),
    gitOutput(pi, cwd, ["branch", "--show-current"]),
    gitOutput(pi, cwd, ["rev-parse", "HEAD"]),
  ]);
  return { worktreeRoot, currentBranch, headSHA };
}

async function gitOutput(pi, cwd, args) {
  try {
    const result = await pi.exec("git", ["-C", cwd, ...args], { timeout: 5000 });
    if (result.code !== 0) return "";
    return result.stdout.trim();
  } catch {
    return "";
  }
}

function bridgeRuntimeDir() {
  const base = process.env.XDG_RUNTIME_DIR || process.env.XDG_CACHE_HOME || join(homedir(), ".cache");
  const dir = join(base, process.env.XDG_RUNTIME_DIR ? BRIDGE_DIR : join("ero", "runtime", BRIDGE_DIR));
  mkdirSync(dir, { recursive: true, mode: 0o700 });
  chmodSync(dir, 0o700);
  return dir;
}

function readState(path) {
  try {
    return JSON.parse(readFileSync(path, "utf8"));
  } catch {
    return { sessions: [] };
  }
}

function writeState(path, state) {
  writeFileSync(path, JSON.stringify(state, null, 2), { mode: 0o600 });
  chmodSync(path, 0o600);
}

function upsertSession(path, session) {
  const state = readState(path);
  const sessions = state.sessions.filter((item) => item.session_id !== session.session_id);
  sessions.push(session);
  writeState(path, { sessions });
}

function removeSession(path, sessionId) {
  const state = readState(path);
  writeState(path, { sessions: state.sessions.filter((item) => item.session_id !== sessionId) });
}
