import os
import subprocess

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, '..', '..', '..')) + '/'
WORK = os.environ.get('PKGSPLIT_WORK', '/tmp/pkgsplit') + '/'
BIN = os.environ.get('PKGSPLIT_BIN', WORK + 'bin') + '/'
import sys


root = sys.argv[1]
files = []
for line in open(sys.argv[2]):
    f = line.strip()
    if not f.endswith(".go"):
        continue
    p = os.path.join(root, f)
    if not os.path.exists(p):
        continue
    with open(p, "rb") as fh:
        if b"DO NOT EDIT" in fh.read(3000):
            continue
    files.append(p)
subprocess.run(["python3", "-I", HERE + "/normimports.py", *files], check=True)
for i in range(0, len(files), 200):
    subprocess.run(["golines", "-m", "100", "-w", *files[i : i + 200]], check=True)
print("formatted", len(files))
