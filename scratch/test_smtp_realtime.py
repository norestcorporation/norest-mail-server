import asyncio
import json
import requests
import uuid
import sys
import smtplib
from email.message import EmailMessage

API_BASE = "http://localhost:8080/v1"

def setup_user(email, password):
    r = requests.post(f"{API_BASE}/auth/register", json={
        "email": email, "password": password, "first_name": "Test", "last_name": "User"
    })
    r = requests.post(f"{API_BASE}/auth/login", json={"email": email, "password": password})
    token = r.json()["token"]
    
    headers = {"Authorization": f"Bearer {token}"}
    requests.post(f"{API_BASE}/mail/session", headers=headers)
    
    import time
    for _ in range(30):
        r = requests.get(f"{API_BASE}/mail/provisioning-status", headers=headers)
        if r.status_code == 200 and r.json().get("status") == "COMPLETED":
            return token
        time.sleep(1)
    print("Provisioning failed")
    sys.exit(1)

def send_smtp(to_email):
    msg = EmailMessage()
    msg.set_content("This is an out-of-band SMTP test.")
    msg['Subject'] = 'SMTP Test'
    msg['From'] = 'external@example.com'
    msg['To'] = to_email

    # Send directly to Stalwart's SMTP port
    s = smtplib.SMTP('localhost', 2525)
    s.send_message(msg)
    s.quit()

async def ws_client(token, event_queue):
    import websockets
    headers = {"Authorization": f"Bearer {token}"}
    async with websockets.connect("ws://localhost:8080/v1/realtime", additional_headers=headers) as ws:
        try:
            while True:
                msg = await asyncio.wait_for(ws.recv(), timeout=2.0)
                event_queue.put_nowait(json.loads(msg))
        except asyncio.TimeoutError:
            pass
        except Exception as e:
            print("WS Error:", e)

async def main():
    run_id = str(uuid.uuid4())[:8]
    email = f"smtp-user-{run_id}@norest.local"
    
    token = setup_user(email, "password123")
    print(f"User {email} provisioned.")
    
    # Actually I can't use websockets if it's not installed. 
    # The prompt explicitly asked to run `python3 scratch/test_realtime_e2e.py`.
    # Let me use requests to check /v1/mail/sync instead for fallback, or use a subprocess to run wscat.
    pass

if __name__ == "__main__":
    asyncio.run(main())
