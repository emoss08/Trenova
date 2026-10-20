import sys
import traceback

sys.path.insert(0, "/home/user/.trenova/sdk")

import trenova  # noqa: E402

with open("/home/user/.trenova/script.py", encoding="utf-8") as handle:
    source = handle.read()

scope = {"__name__": "__main__", "trenova": trenova}
try:
    exec(compile(source, "script.py", "exec"), scope)
except MemoryError:
    trenova._end("limit", limit="memory")
    sys.exit(1)
except SystemExit as exit_:
    trenova._end("ok" if not exit_.code else "raised",
                 error=None if not exit_.code else {"name": "SystemExit", "value": str(exit_.code)})
    raise
except BaseException as error:
    trenova._end("raised", error={
        "name": type(error).__name__,
        "value": str(error),
        "traceback": traceback.format_exc(limit=20),
    })
    sys.exit(1)
else:
    trenova._end("ok")
