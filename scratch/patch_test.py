import re

with open("scratch/test_realtime_e2e.py", "r") as f:
    content = f.read()

content = content.replace('token = json.loads(r.read())["token"]', 'resp = json.loads(r.read()); print(resp); token = resp["token"]')

with open("scratch/test_realtime_e2e.py", "w") as f:
    f.write(content)
