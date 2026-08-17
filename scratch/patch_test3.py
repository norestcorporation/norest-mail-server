import re

with open("scratch/test_realtime_e2e.py", "r") as f:
    content = f.read()

# Remove the lines calling /mail/session
content = re.sub(
    r"req = urllib\.request\.Request\(f\"\{API_BASE\}/mail/session\".*?\n\s*urllib\.request\.urlopen\(req\)\n",
    "",
    content
)

with open("scratch/test_realtime_e2e.py", "w") as f:
    f.write(content)
