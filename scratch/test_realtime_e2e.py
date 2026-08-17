import urllib.request
import json
import uuid
import sys
import smtplib
from email.message import EmailMessage
import time
import socket
import base64
import os

API_BASE = "http://localhost:8080/v1"

def setup_user(email, password):
    req = urllib.request.Request(f"{API_BASE}/auth/register", data=json.dumps({"email": email, "password": password, "first_name": "Test", "last_name": "User"}).encode(), headers={'Content-Type': 'application/json'})
    try:
        urllib.request.urlopen(req)
    except urllib.error.HTTPError as e:
        if e.code not in (200, 201, 409):
            print(f"Failed to register {email}: {e.read()}")
            sys.exit(1)
            
    req = urllib.request.Request(f"{API_BASE}/auth/login", data=json.dumps({"email": email, "password": password}).encode(), headers={'Content-Type': 'application/json'})
    r = urllib.request.urlopen(req)
    resp = json.loads(r.read()); print(resp); token = resp["access_token"]
    
        
    for _ in range(30):
        req = urllib.request.Request(f"{API_BASE}/mail/provisioning-status", headers={"Authorization": f"Bearer {token}"})
        r = urllib.request.urlopen(req)
        if json.loads(r.read()).get("status") == "COMPLETED":
            return token
        time.sleep(1)
    print("Provisioning failed")
    sys.exit(1)

def send_smtp(to_email, subject):
    msg = EmailMessage()
    msg.set_content("This is an out-of-band SMTP test.")
    msg['Subject'] = subject
    msg['From'] = 'external@example.com'
    msg['To'] = to_email
    s = smtplib.SMTP('localhost', 2525)
    s.send_message(msg)
    s.quit()

def ws_connect_and_wait(token, expected_type):
    # Manual WebSocket handshake
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect(('localhost', 8080))
    
    key = base64.b64encode(os.urandom(16)).decode()
    handshake = (
        f"GET /v1/realtime HTTP/1.1\r\n"
        f"Host: localhost:8080\r\n"
        f"Upgrade: websocket\r\n"
        f"Connection: Upgrade\r\n"
        f"Sec-WebSocket-Key: {key}\r\n"
        f"Sec-WebSocket-Version: 13\r\n"
        f"Authorization: Bearer {token}\r\n\r\n"
    )
    s.sendall(handshake.encode())
    
    resp = b""
    while b"\r\n\r\n" not in resp:
        resp += s.recv(1024)
        
    s.settimeout(20.0) # wait up to 20s for the worker to poll
    try:
        while True:
            # simple ws framing parser
            head = s.recv(2)
            if not head:
                break
            length = head[1] & 127
            if length == 126:
                length = int.from_bytes(s.recv(2), 'big')
            elif length == 127:
                length = int.from_bytes(s.recv(8), 'big')
            
            payload = s.recv(length)
            event = json.loads(payload.decode('utf-8'))
            print("Received event:", event)
            if event.get("event_type") == expected_type:
                return True
    except socket.timeout:
        print("Timeout waiting for WebSocket event")
    except Exception as e:
        print("WS error:", e)
    finally:
        s.close()
    return False

def main():
    run_id = str(uuid.uuid4())[:8]
    email = f"smtp-user-{run_id}@norest.local"
    
    token = setup_user(email, "password123")
    print(f"User {email} provisioned.")
    
    # We will fork a process to read from WS, while we send SMTP
    import threading
    result = []
    def wait_ws():
        res = ws_connect_and_wait(token, "message.created")
        result.append(res)
        
    t = threading.Thread(target=wait_ws)
    t.start()
    
    time.sleep(1) # wait for WS to connect
    
    subject = f"SMTP Test {run_id}"
    print(f"Sending SMTP email: {subject}")
    send_smtp(email, subject)
    
    t.join()
    if not result or not result[0]:
        print("FAIL: Did not receive event")
        sys.exit(1)
        
    print("SUCCESS: SMTP out-of-band delivery triggered realtime event successfully!")

if __name__ == "__main__":
    main()
