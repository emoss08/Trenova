import json
import os
import resource
import signal
import subprocess
import sys
import threading

with open("/home/user/.trenova/config.json", encoding="utf-8") as _handle:
    _config = json.load(_handle)

_nonce = _config["nonce"]


def _limits():
    memory = _config["memoryBytes"]
    cpu = _config["wallSeconds"]
    resource.setrlimit(resource.RLIMIT_AS, (memory, memory))
    resource.setrlimit(resource.RLIMIT_CPU, (cpu, cpu))
    resource.setrlimit(resource.RLIMIT_NPROC, (64, 64))
    resource.setrlimit(resource.RLIMIT_FSIZE, (64 << 20, 64 << 20))
    resource.setrlimit(resource.RLIMIT_NOFILE, (256, 256))
    os.setsid()


def _end(status, limit=""):
    body = json.dumps({"status": status, "limit": limit}, separators=(",", ":"))
    sys.stdout.write("\x1etrenova-end:%s:%s\n" % (_nonce, body))
    sys.stdout.flush()


def _relay(stream, target):
    for line in iter(stream.readline, ""):
        target.write(line)
        target.flush()
    stream.close()


os.makedirs("/home/user/work", exist_ok=True)
_env = {
    "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
    "HOME": "/home/user/work",
    "OPENBLAS_NUM_THREADS": "1",
    "OMP_NUM_THREADS": "1",
    "MKL_NUM_THREADS": "1",
    "MPLBACKEND": "Agg",
}
_child = subprocess.Popen(
    [sys.executable, "-I", "-u", "/home/user/.trenova/sdk/main.py"],
    cwd="/home/user/work",
    env=_env,
    stdin=subprocess.DEVNULL,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    text=True,
    bufsize=1,
    preexec_fn=_limits,
)
_out = threading.Thread(target=_relay, args=(_child.stdout, sys.stdout), daemon=True)
_err = threading.Thread(target=_relay, args=(_child.stderr, sys.stderr), daemon=True)
_out.start()
_err.start()
try:
    _code = _child.wait(timeout=_config["wallSeconds"])
except subprocess.TimeoutExpired:
    os.killpg(_child.pid, signal.SIGKILL)
    _child.wait()
    _out.join(timeout=2)
    _err.join(timeout=2)
    _end("killed", "wall")
else:
    _out.join(timeout=2)
    _err.join(timeout=2)
    if _code == -signal.SIGXCPU or _code == -signal.SIGKILL:
        _end("killed", "cpu" if _code == -signal.SIGXCPU else "memory")
    elif _code < 0:
        _end("killed", "")
