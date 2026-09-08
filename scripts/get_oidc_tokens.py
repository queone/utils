#!/usr/bin/env python
# get_oidc_tokens.py 1.0.2
# See https://que.one/git/oidc-azure.html

import os
import sys
import requests
import msal
import base64
import json
import datetime

# Global constants and variables
BLUE      = Blu = "\033[1;34m"
CYAN      = Cya = "\033[1;36m"
GREEN     = Grn = "\033[1;32m"
LIGHTGRAY = Gra = "\033[37m"
MAGENTA   = Mag = "\033[1;35m"
RED       = Red = "\033[1;31m"
WHITE     = Whi = "\033[1;37m"
YELLOW    = Yel = "\033[1;33m"
RESET     = Rst = "\033[0m"

# Github actions caller MUST define below variables
CLIENT_ID         = os.getenv("CLIENT_ID")  # Client ID of the Azure AD app
TENANT_ID         = os.getenv("TENANT_ID")  # Tenant ID of the Azure AD app
gh_oidc_req_token = os.getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
gh_oidc_req_url   = os.getenv("ACTIONS_ID_TOKEN_REQUEST_URL")

# Standard OAuth 2.0 Scopes for Microsoft Azure Services. These are used to
# request tokens for accessing the specific service API with default permissions.
AZ_SCOPES    = ["https://management.azure.com/.default"]               # arm:       Azure Resource Manager
MG_SCOPES    = ["https://graph.microsoft.com/.default"]                # ms-graph:  MS Graph
AAD_SCOPES   = ["https://graph.windows.net/.default"]                  # aad-graph: Azure AD Graph API
BATCH_SCOPES = ["https://batch.core.windows.net/.default"]             # batch:     Azure Batch
DLS_SCOPES   = ["https://datalake.azure.net/.default"]                 # data-lake: Azure Data Lake Storage
MEDIA_SCOPES = ["https://rest.media.azure.net/.default"]               # media:     Azure Media Services
SQL_SCOPES   = ["https://ossrdbms-aad.database.windows.net/.default"]  # oss-rdbms: Azure Database for PostgreSQL/MySQL

def printc(color, *args, **kwargs):
    valid_colors = {BLUE, CYAN, GREEN, LIGHTGREY, MAGENTA, RED, WHITE, YELLOW}
    if color not in valid_colors:
        print(f"Error: Invalid color constant. Available colors: BLUE, CYAN, "
              "GREEN, LIGHTGREY, MAGENTA, RED, WHITE, YELLOW")
        return
    print(color + ' '.join(map(str, args)) + RESET, **kwargs)

def decode_oidc_token(oidc_token):
    try:
        parts = oidc_token.split(".")  # Split the token into its three parts
        if len(parts) != 3:
            raise ValueError("Invalid OIDC token format")
        payload_encoded = parts[1]     # Extract the payload (second part)
        # Add padding to the payload if necessary (Base64 requires padding)
        padding = len(payload_encoded) % 4
        if padding > 0:
            payload_encoded += "=" * (4 - padding)
        # Decode the Base64-encoded payload
        payload_decoded = base64.b64decode(payload_encoded).decode("utf-8")
        payload = json.loads(payload_decoded)  # Parse the payload as JSON
        return payload
    except Exception as e:
        raise ValueError(f"Failed to decode OIDC token: {str(e)}")

def get_github_oidc_token():
    # Validate the special Github variables
    if not gh_oidc_req_token:
        printc(RED, "==> Error. Variable 'ACTIONS_ID_TOKEN_REQUEST_TOKEN' is empty or null")
        print("Have you enabled right workflow permissions?")
        sys.exit(1)
    if not gh_oidc_req_url:
        printc(RED, "==> Error. Variable 'ACTIONS_ID_TOKEN_REQUEST_URL' is empty or null")
        print("Have you enabled right workflow permissions?")
        sys.exit(1)
    printc(YELLOW, f"==> ACTIONS_ID_TOKEN_REQUEST_TOKEN Hint = {gh_oidc_req_token[0:4]}******")
    printc(YELLOW, f"==> ACTIONS_ID_TOKEN_REQUEST_URL        = {gh_oidc_req_url}")

    # Fetch Github OIDC token
    headers = {
        "Authorization": f"Bearer {gh_oidc_req_token}",
        "Accept": "application/json"
    }
    # Add the audience parameter to the request URL
    params = {
        "audience": "api://AzureADTokenExchange"  # Explicitly set the audience
        # Could this be parameterized? To say, match a specific Github repo, and
        # the specific configuration of the Azure App registration federated
        # credential 'Audience' setting?
    }
    response = requests.get(gh_oidc_req_url, headers=headers, params=params)
    if response.status_code == 200:
        oidc_token = response.json().get("value")  # Extract the OIDC token
    else:
        raise Exception(f"Failed to fetch OIDC token: {response.status_code} - {response.text}")

    # Print useful debugging info about this token 
    printc(YELLOW, f"==> Github OIDC_TOKEN Hint = {oidc_token[0:4]}******")
    printc(YELLOW, f"==> Decoded Github OIDC_TOKEN:")
    decoded_token = decode_oidc_token(oidc_token)
    for key, value in decoded_token.items():
        if key in ['aud', 'iss', 'sub']:
            # Highlight these in yellow
            printc(YELLOW, f"    {key}: {value}")
        elif key in ['exp', 'nbf', 'iat']:
            # Highlight these in human readable format and green
            v = datetime.datetime.fromtimestamp(value).strftime('%Y-%b-%d %H:%M')
            printc(GREEN, f"    {key}: {v} ({value})")
        else:
            print(f"    {key}: {value}")
    return oidc_token

def get_microsoft_token(oidc_token, scopes):
    # Create a confidential client application
    app = msal.ConfidentialClientApplication(
        client_id=CLIENT_ID,
        client_credential={"client_assertion": oidc_token},  # Use OIDC token as client assertion
        authority=f"https://login.microsoftonline.com/{TENANT_ID}"
    )

    # Exchange the OIDC token for an Azure access token
    result = app.acquire_token_for_client(scopes=scopes)

    if "access_token" in result:
        return result["access_token"]
    else:
        raise Exception("Failed to acquire access token: " + str(result.get("error_description")))

def set_api_token(oidc_token, scopes, env_var_name):
    """
    Obtain an API token for the specified scopes and set it as an environment variable.

    :param oidc_token: The OIDC token obtained from GitHub.
    :param scopes: The scopes for which to request the token.
    :param env_var_name: The name of the environment variable to set.
    """
    global ERROR_ENCOUNTERED
    try:
        # Get the token for the specified scopes
        api_token = get_microsoft_token(oidc_token, scopes)
        printc(YELLOW, f"==> {env_var_name} Hint = {api_token[0:4]}******")
        
        # Set the token as an environment variable
        with open(os.getenv("GITHUB_ENV"), "a") as f:
            f.write(f"{env_var_name}={api_token}\n")
    except Exception as e:
        printc(RED, f"==> Error setting {env_var_name}: {str(e)}")
        ERROR_ENCOUNTERED = True

def main():
    global ERROR_ENCOUNTERED
    ERROR_ENCOUNTERED = False
    oidc_token = get_github_oidc_token()  # Fetch the Github OIDC_TOKEN
    set_api_token(oidc_token, AZ_SCOPES, 'AZ_TOKEN')  # Set the ARM API AZ token
    set_api_token(oidc_token, MG_SCOPES, 'MG_TOKEN')  # Set the MS Graph API token
    # You can do additional Microsoft Azure Services API tokens below ...

    if ERROR_ENCOUNTERED:
        print(f"{Red}==> There were errors acquiring and setting the tokens.{Rst}")

if __name__ == "__main__":
    main()
