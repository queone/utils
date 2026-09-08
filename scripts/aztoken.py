# aztoken.py

import sys
import time
import os
import json
from datetime import datetime, timedelta
import signal
import msal

BLUE = '\x1b[1;34m'
GREEN = '\x1b[32m'
RED = '\x1b[31m'
RESET = '\x1b[0m'

cache = msal.TokenCache()   # Initialize a global token cache

# Quick exit on CTRL-C
def exit_gracefully(signal, frame):
    sys.exit(0)
signal.signal(signal.SIGTERM, exit_gracefully)
signal.signal(signal.SIGINT, exit_gracefully)

def print_flush(message):
    print(message)
    sys.stdout.flush()

def expiry_date(expires_in_seconds):
    current_time = datetime.now()
    if expires_in_seconds == None:
        expires_in_seconds = 0
    expiry_date_temp = current_time + timedelta(seconds=expires_in_seconds)
    return expiry_date_temp.strftime('%Y-%m-%d %H:%M:%S')

def get_token_by_credentials(scopes, client_id, client_secret, authority_url):
    # Define the client application using MSAL
    cca = msal.ConfidentialClientApplication(
        client_id,
        authority=authority_url,
        client_credential=client_secret,
        token_cache=cache  # Use the global cache
    )

    # Acquire a token using client credentials
    token_request = {
        'scopes': scopes
    }

    try:
        result = cca.acquire_token_for_client(scopes=scopes)
        return result
    except Exception as error:
        raise Exception(f"Error acquiring token: {str(error)}")

def main():
    # scopes = ['https://graph.microsoft.com/.default']
    # scopes = ['https://management.azure.com/.default']
    scopes = ['https://ossrdbms-aad.database.windows.net/.default']
    client_id = os.environ.get('MAZ_CLIENT_ID')
    client_secret = os.environ.get('MAZ_CLIENT_SECRET')
    tenant_id = os.environ.get('MAZ_TENANT_ID')
    authority_url = f'https://login.microsoftonline.com/{tenant_id}'

    while True:
        token = get_token_by_credentials(scopes, client_id, client_secret, authority_url)
        if 'access_token' not in token:
            print_flush(f"{RED}Failed to obtain token: {token}{RESET}")
        #print(json.dumps(token, indent=2))  # OPTION: Print entire token structure

        access_token = token.get('access_token')
        expires_in_secs = token.get('expires_in')
        expires_in = expiry_date(expires_in_secs)

        print_flush(f"\n{BLUE}TOKEN DETAILS{RESET}:")
        print_flush(f"{BLUE}  client_id{RESET} : {GREEN}{client_id}{RESET}")
        print_flush(f"{BLUE}  Authority{RESET} : {GREEN}{authority_url}{RESET}")
        print_flush(f"{BLUE}  Scopes{RESET}    : {GREEN}{scopes}{RESET}")
        print_flush(f"{BLUE}  Token{RESET}     : {GREEN}{access_token}{RESET}")
        print_flush(f"{BLUE}  Expires On{RESET}: {GREEN}{expires_in}{RESET} ({expires_in_secs} seconds)")

        # Wait for 5 seconds (5,000 milliseconds) before making the next call
        time.sleep(5)

if __name__ == "__main__":
    main()
