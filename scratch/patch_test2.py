import re

with open("scratch/test_realtime_e2e.py", "r") as f:
    content = f.read()

content = content.replace('token = resp["token"]', 'token = resp["access_token"]')

with open("scratch/test_realtime_e2e.py", "w") as f:
    f.write(content)
